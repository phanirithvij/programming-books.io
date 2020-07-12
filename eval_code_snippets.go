package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/u"

	"github.com/kjk/notionapi"
)

func setDefaultFileNameFromLanguage(sf *SourceFile) error {
	if sf.Directive.FileName != "" {
		return nil
	}

	// we don't care unless it goes to glot.io
	if !sf.Directive.Glot {
		return nil
	}

	ext := ""
	lang := strings.ToLower(sf.Lang)
	switch lang {
	case "go":
		ext = ".go"
	case "javascript":
		ext = ".js"
	case "cpp", "cplusplus", "c++":
		ext = ".cpp"
	default:
		logf("detectFileNameFromLanguage: lang '%s' is not supported\n", sf.Lang)
		logf("Notion page: %s\n", sf.NotionOriginURL)
		panic("")
	}
	sf.Directive.FileName = "main" + ext
	if sf.FileName == "" {
		sf.FileName = sf.Directive.FileName
		sf.Path = sf.FileName
	}
	return nil
}

type CodeEvalInfo struct {
	URL    string
	GistID string
	// TODO: maybe more info, like which file to show if more than one
	// alternatively this might come from codeeval.yml
}

// returns empty string if s is not code eval url
func parseCodeEvalGist(s string) string {
	if !strings.Contains(s, "https://codeeval.dev/gist/") {
		return ""
	}
	gist := strings.TrimPrefix(s, "https://codeeval.dev/gist/")
	panicIf(len(gist) != len("5648d36550f7592236327e920243f791"), "'%s' doesn't look like a valid gist in url '%s'", gist, s)
	return gist
}

// parses a line like:
// https://codeeval.dev/gist/5648d36550f7592236327e920243f791 main.go
// return nil if is not code eval line
func parseCodeEvalInfo(s string) *CodeEvalInfo {
	if !strings.Contains(s, "https://codeeval.dev/gist/") {
		return nil
	}
	res := &CodeEvalInfo{}
	parts := strings.Split(s, " ")
	for _, uri := range parts {
		gistID := parseCodeEvalGist(uri)
		if gistID == "" {
			continue
		}
		panicIf(res.GistID != "", "has second uri: '%s', first: '%s'", uri, res.URL)
		res.GistID = gistID
		res.URL = s
	}
	return res
}

func gistDownloadCached(cache *Cache, gistID string) string {
	gist := cache.getGistByID(gistID)
	if gist != nil && !flgGistRedownload {
		logVerbose("gist '%s': got from cache\n", gistID)
		return gist.Gist
	}
	timeStart := time.Now()
	newGist := gistDownloadMust(gistID)
	logf("gist '%s': downloaded in %s\n", gistID, time.Since(timeStart))
	if gist != nil && newGist.Raw == gist.Gist {
		panicIf(!flgGistRedownload)
		return newGist.Raw
	}
	cache.saveGist(gistID, newGist.Raw)
	return newGist.Raw
}

func langFromFileName(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return "go"
	case ".cpp", ".cc", ".cxx", ".c":
		return "gcc"
	case ".js":
		return "node"
	default:
		panic(fmt.Sprintf("Unsupported extensions '%s'", ext))
	}

}
func langFromGist(gist *Gist) string {
	var currLang string
	for _, f := range gist.Files {
		lang := langFromFileName(f.Filename)
		if lang == "" {
			continue
		}
		if currLang == "" {
			currLang = lang
			continue
		}
		panicIf(lang != currLang, "lang:")
	}
	panicIf(currLang == "")
	return currLang
}

func getEvalResponseString(resp *EvalResponse) string {
	return resp.Stdout + resp.Stderr + resp.Error
}

func evalGist(gistStr string) (*EvalResponse, string) {
	gist := gistDecode(gistStr)
	panicIf(gist.Truncated) // TODO: implement if needed
	var files []File
	for name, gf := range gist.Files {
		// logf("  file: '%s'\n", name)
		panicIf(gf.Truncated)
		f := File{
			Name:    name,
			Content: gf.Content,
		}
		files = append(files, f)
	}

	lang := langFromGist(gist)
	// TODO: set the language based on files or what Gist sets
	e := &Eval{
		Language: lang,
		Files:    files,
	}
	resp, err := evalGo(e)
	must(err)
	out := getEvalResponseString(resp)
	// logf("Eval response:\n%s\n", out)
	return resp, out
}

func evalCached(cache *Cache, gist string) string {
	sha1 := u.Sha1HexOfBytes([]byte(gist))
	gistOut := cache.getGistOuputBySha1(sha1)
	if gistOut != nil {
		return gistOut.Output
	}
	_, out := evalGist(gist)
	cache.saveGistOutput(gist, out)
	return out
}

func getGistFile(gist *Gist) *GistFile {
	// TODO: support multiple files
	panicIf(len(gist.Files) != 1, "TODO: only supporting a single file for now")
	for _, f := range gist.Files {
		panicIf(f.Truncated)
		return f
	}
	panic("not reachable")
}

func evalCodeEval(page *Page, block *notionapi.Block, gistInfo string) {
	info := parseCodeEvalInfo(gistInfo)
	gistStr := gistDownloadCached(page.Book.cache, info.GistID)

	output := evalCached(page.Book.cache, gistStr)
	gist := gistDecode(gistStr)
	panicIf(gist.Truncated) // TODO: implement if needed
	gistFile := getGistFile(gist)
	lang := langFromGist(gist)
	sf := &SourceFile{
		NotionOriginURL: fmt.Sprintf("https://notion.so/%s", toNoDashID(page.NotionID)),
		//Path:      path,
		FileName:   gistFile.Filename,
		Lang:       lang,
		GlotOutput: output,
	}

	sf.SnippetName = page.PageTitle()
	if sf.SnippetName == "" {
		sf.SnippetName = "untitled"
	}

	sf.PlaygroundURI = "https://codeeval.dev/gist/" + info.GistID

	data := []byte(gistFile.Content)
	err := setSourceFileData(sf, data)
	must(err)

	if page.blockCodeToSourceFile == nil {
		page.blockCodeToSourceFile = map[string]*SourceFile{}
	}
	page.blockCodeToSourceFile[block.ID] = sf
}

func evalCodeSnippetsForPage(page *Page) {
	book := page.Book
	// can happen for draft pages
	if book == nil {
		return
	}
	fnText := func(block *notionapi.Block) {
		panicIf(block.Type != notionapi.BlockText)
		ts := block.GetTitle()
		s := notionapi.TextSpansToString(ts)
		if !strings.Contains(s, "https://codeeval.dev/gist/") {
			return
		}
		evalCodeEval(page, block, s)
	}

	fnEmbed := func(block *notionapi.Block) {
		panicIf(block.Type != notionapi.BlockEmbed)
		uri := block.FormatEmbed().DisplaySource
		if strings.Contains(uri, "https://codeeval.dev") {
			logf("found codeeval: '%s'\n", uri)
			panic("embed blocks NYI")
		}
	}

	fn := func(block *notionapi.Block) {
		if block.Type == notionapi.BlockText {
			fnText(block)
			return
		}

		if block.Type == notionapi.BlockEmbed {
			fnEmbed(block)
			return
		}

		if block.Type != notionapi.BlockCode {
			return
		}

		if false {
			s := block.Code
			lines := dataToLines([]byte(s))
			firstLine := ""
			if len(lines) > 0 {
				firstLine = lines[0]
			}
			logf("Page: %s\n  %s\n", page.Title, firstLine)
		}

		//lang := getLangFromFileExt(filepath.Ext(path))
		lang := block.CodeLanguage
		sf := &SourceFile{
			NotionOriginURL: fmt.Sprintf("https://notion.so/%s", toNoDashID(page.NotionID)),
			//Path:      path,
			//FileName:  name,
		}
		sf.Lang = lang
		sf.SnippetName = page.PageTitle()
		if sf.SnippetName == "" {
			sf.SnippetName = "untitled"
		}

		data := []byte(block.Code)
		err := setSourceFileData(sf, data)
		if err != nil {
			logf("genBlock: setSourceFileData() failed with '%s'\n", err)
			logf("page: %s\n", sf.NotionOriginURL)
			//must(err)
		}

		if sf.Directive.Glot {
			logVerbose("!code glot %s\n", sf.NotionOriginURL)
			createGistFromGlot(sf)
			// for those we respect no output/no playground
		} else {
			//logVerbose("!code no glot %s\n", sf.NotionOriginURL)
			// for embedded code blocks by default we don't set playground
			// or output unless explicitly asked for
			sf.Directive.NoPlayground = true
			sf.Directive.NoOutput = true
		}
		setDefaultFileNameFromLanguage(sf)

		if page.blockCodeToSourceFile == nil {
			page.blockCodeToSourceFile = map[string]*SourceFile{}
		}
		page.blockCodeToSourceFile[block.ID] = sf
	}

	page.NotionPage.ForEachBlock(fn)
}

// TODO: we might do some re-use
func createGistFromGlot(sf *SourceFile) {
	panic("shouldn't happen anymore")

	/*
		description := "example for " + sf.NotionOriginURL
		newGist := &GistCreate{
			Description: description,
			Public:      true,
			Files:       map[string]*GistNewFile{},
		}
		file := &GistNewFile{
			Content: sf.CodeFull,
		}
		newGist.Files[fileName] = file
		gist := createGistMust(newGist)
		//gistURL := "https://gist.github.com/" + gist.ID
		codeEvalURL := "https://codeeval.dev/gist/" + gist.ID
		logf("Created a gist %s\n%s\n\n", sf.NotionOriginURL, codeEvalURL)
	*/
}
