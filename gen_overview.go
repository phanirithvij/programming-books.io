package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kjk/notionapi"
)

var htmlOverview = `<!doctype html>
<html lang="en">
<body>
%s
</body>
</html>
`

func genOverviewPageLink(page *Page) string {
	title := strings.TrimSpace(page.Title)
	uri := page.CanonnicalURL()
	return fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, uri, title)
}

func genOverviewPageCodeBlocks(page *Page) []string {
	var res []string
	cb := func(block *notionapi.Block) {
		if block.Type == notionapi.BlockCode {
			s := block.Code
			lines := dataToLines([]byte(s))
			nLines := len(lines)
			s = fmt.Sprintf(`code %s, %d lines`, block.CodeLanguage, nLines)
			res = append(res, "<li>", s, "</li>")
			return
		}
		if block.Type == notionapi.BlockText {
			ts := block.GetTitle()
			s := notionapi.TextSpansToString(ts)
			if !strings.Contains(s, "https://codeeval.dev/") {
				return
			}
			parts := strings.SplitN(s, " ", 2)
			uri := parts[0]
			s = fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, uri, uri)
			if len(parts) > 1 {
				s += " " + parts[1]
			}
			res = append(res, "<li>", s, "</li>")
			return
		}
	}
	page.NotionPage.ForEachBlock(cb)
	return res
}

func skipURL(uri string) bool {
	if strings.Contains(uri, "notion.so/") {
		return true
	}
	if strings.Contains(uri, "codeeval.dev/") {
		return true
	}
	if strings.Contains(uri, "stackoverflow.com/users/") {
		return true
	}
	return false
}

func isStackOverflowURL(uri string) bool {
	return strings.Contains(uri, "stackoverflow.com/") && !strings.Contains(uri, "/users/")
}

func genExternalLinksInPage(page *notionapi.Page) []string {
	var res []string
	rememberLink := func(uri string) {
		if skipURL(uri) {
			return
		}
		if isStackOverflowURL(uri) {
			s := fmt.Sprintf(`<li>so link: %s</li>`, uri)
			res = append(res, s)
			return
		}
		// TODO: report those links or not?
		s := fmt.Sprintf(`<li>link: %s</li>`, uri)
		res = append(res, s)
	}

	findLinks := func(b *notionapi.Block) {
		spans := b.GetTitle()
		for _, ts := range spans {
			for _, attr := range ts.Attrs {
				attrType := notionapi.AttrGetType(attr)
				if attrType == notionapi.AttrLink {
					uri := notionapi.AttrGetLink(attr)
					rememberLink(uri)
				}
			}
		}
	}

	page.ForEachBlock(findLinks)
	return res
}

func genOverviewPage(page *Page) []string {
	var lines []string
	s := genOverviewPageLink(page)
	lines = append(lines, "<li>", s, "</li>")
	lines = append(lines, "<ul>")
	lines = append(lines, genOverviewPageCodeBlocks(page)...)
	lines = append(lines, genExternalLinksInPage(page.NotionPage)...)
	lines = append(lines, "</ul>")
	return lines
}

// generates overview.html which shows the structure of the html
func genOverviewContent(book *Book) []byte {
	lines := []string{}
	lines = append(lines, "<ul>")
	for _, chapter := range book.Chapters() {
		lines = append(lines, genOverviewPage(chapter)...)

		lines = append(lines, "<ul>")
		for _, page := range chapter.Pages {
			lines = append(lines, genOverviewPage(page)...)
		}
		lines = append(lines, "</ul>")
	}
	lines = append(lines, "</ul>")
	str := strings.Join(lines, "\n")
	str = fmt.Sprintf(htmlOverview, str)
	return []byte(str)
}

func genOverview(book *Book) {
	d := genOverviewContent(book)
	path := filepath.Join(book.DirOnDisk, "overview.html")
	writeFileMust(path, d)
}
