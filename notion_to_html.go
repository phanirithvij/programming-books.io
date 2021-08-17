package main

import (
	"bytes"
	"html"
	"strings"

	"github.com/kjk/notionapi"
	"github.com/kjk/notionapi/tohtml"

	"github.com/kjk/u"
)

/*
Todo:
- improve style of .img class (take from notionapi)
- set the right margin-bottom to .title
*/

// Converter is for notion -> HTML generation
type Converter struct {
	page *Page
	book *Book

	converter *tohtml.Converter
}

func (c *Converter) reportIfInvalidLink(uri string, extractedID string) {
	pageID := c.page.getID()
	logf("Found invalid link '%s' (id: '%s') in page https://notion.so/%s\n", uri, extractedID, pageID)
	if extractedID == "" {
		return
	}
	page := findPageByID(c.book, extractedID)
	if page != nil {
		logf(" strange, we actually found it via findPageByID()\n")
	}
}

// change https://www.notion.so/Advanced-web-spidering-with-Puppeteer-ea07db1b9bff415ab180b0525f3898f6
// =>
// url within the book
func (c *Converter) rewriteURL(uri string) string {
	if !strings.Contains(uri, "notion.so/") {
		return uri
	}

	id := notionapi.ExtractNoDashIDFromNotionURL(uri)
	if id == "" {
		c.reportIfInvalidLink(uri, id)
		return uri
	}
	page := c.book.idToPage[id]
	if page == nil {
		c.reportIfInvalidLink(uri, id)
		return uri
	}
	page.Book = c.book
	return page.URL()
}

func (c *Converter) getURLAndTitleForBlock(block *notionapi.Block) (string, string) {
	id := toNoDashID(block.ID)
	page := c.book.idToPage[id]
	if page == nil {
		title := cleanTitle(block.Title)
		logf("No article for id %s %s\n", id, title)
		url := "/article/" + id + "/" + urlify(title)
		return url, title
	}

	return page.URL(), page.Title
}

func findPageByID(book *Book, id string) *Page {
	pages := book.GetAllPages()
	for _, page := range pages {
		if strings.EqualFold(page.getID(), id) {
			return page
		}
	}
	return nil
}

func findImageMapping(images []*ImageMapping, link string) *ImageMapping {
	for _, im := range images {
		if im.link == link {
			return im
		}
	}
	logf("Didn't find image with link '%s'\n", link)
	logf("Available images:\n")
	for _, im := range images {
		logf("  link: %s, relativeURL: %s, path: %s\n", im.link, im.relativeURL, im.path)
	}
	return nil
}

// RenderEmbed renders BlockEmbed
// TODO: also check for
// sf := c.page.blockCodeToSourceFile[block.ID]
// c.genSourceFile(sf)
func (c *Converter) RenderEmbed(block *notionapi.Block) bool {
	uri := block.FormatEmbed().DisplaySource
	if strings.Contains(uri, "onlinetool.io/") {
		panic("not supported anymore")
		//c.genGitEmbed(block)
		//return true
	}
	if strings.Contains(uri, "repl.it/") {
		panic("not supported anymore")
		//c.genReplitEmbed(block)
		//return true
	}
	u.PanicIf(true, "unsupported embed %s", uri)
	return false
}

/*
func (c *Converter) genReplitEmbed(block *notionapi.Block) {
	uri := block.FormatEmbed().DisplaySource
	uri = strings.Replace(uri, "?lite=true", "", -1)
	logf("Page: https://notion.so/%s\n", c.page.NotionID)
	logf("  Replit: %s\n", uri)
	panic("we no longer use replit")
}
*/

func (c *Converter) genSourceFile(sf *SourceFile) {
	{
		var tmp bytes.Buffer
		code := sf.CodeToShow()
		lang := sf.Lang
		htmlHighlight(&tmp, string(code), lang, "")
		d := tmp.Bytes()
		info := CodeBlockInfo{
			Lang: sf.Lang,
		}
		info.PlaygroundURI = sf.PlaygroundURI
		s := fixupHTMLCodeBlock(string(d), &info)
		c.converter.Printf(s)
	}

	output := sf.Output()
	if len(output) != 0 {
		var tmp bytes.Buffer
		htmlHighlight(&tmp, output, "text", "")
		d := tmp.Bytes()
		info := CodeBlockInfo{
			Lang: "output",
		}
		s := fixupHTMLCodeBlock(string(d), &info)
		c.converter.Printf(s)
	}
}

func (c *Converter) genGitEmbed(block *notionapi.Block) {
	uri := block.FormatEmbed().DisplaySource
	f := findSourceFileForEmbedURL(c.page, uri)
	// currently we only handle source code file embeds but might handle
	// others (graphs etc.)
	if f == nil {
		logf("genEmbed: didn't find source file for url %s\n", uri)
		return
	}

	c.genSourceFile(f)
}

// RenderCode renders BlockCode
func (c *Converter) RenderCode(block *notionapi.Block) bool {

	sf := c.page.blockCodeToSourceFile[block.ID]
	c.genSourceFile(sf)

	if false {
		// code := html.EscapeString(block.Code)
		//fmt.Fprintf(g.f, `<div class="%s">Lang for code: %s</div>
		//<pre class="%s">
		//%s
		//</pre>`, levelCls, block.CodeLanguage, levelCls, code)
		var tmp bytes.Buffer
		htmlHighlight(&tmp, string(block.Code), block.CodeLanguage, "")
		d := tmp.Bytes()
		var info CodeBlockInfo
		// TODO: set Lang and PlaygroundURI
		s := fixupHTMLCodeBlock(string(d), &info)
		c.converter.Printf(s)
	}
	return true
}

// RenderImage renders BlockImage
// TODO: download images locally like blog
func (c *Converter) RenderImage(block *notionapi.Block) bool {
	link := block.Source
	im := findImageMapping(c.page.Images, link)
	if im != nil {
		link = im.relativeURL
	}
	c.converter.Printf(`<img class="img" src="%s">`, link)
	return true
}

// RenderPage renders BlockPage
func (c *Converter) RenderPage(block *notionapi.Block) bool {
	if c.converter.Page.IsRoot(block) {
		// skips top-level as it's rendered somewhere else
		c.converter.RenderChildren(block)
		return true
	}

	cls := "page-link"
	if block.IsSubPage() {
		cls = "page"
	}

	url, title := c.getURLAndTitleForBlock(block)
	title = html.EscapeString(title)
	c.converter.Printf(`<div class="%s">
<a href="%s">%s</a>`, cls, url, title)
	c.converter.RenderChildren(block)
	c.converter.Printf("`</div>")
	return true
}

// In notion I want to have @TODO lines that are not rendered in html output
func isBlockTextTodo(block *notionapi.Block) bool {
	u.PanicIf(block.Type != notionapi.BlockText, "only supported on '%s' block, called on '%s' block", notionapi.BlockText, block.Type)
	blocks := block.InlineContent
	if len(blocks) == 0 {
		return false
	}
	b := blocks[0]
	// if a block starts with: @TODO, #TODO, :TODO
	// it's a todo block, which we should skip
	if len(b.Text) < 5 {
		return false
	}
	s := strings.ToLower(b.Text[:5])
	switch s {
	case "@todo", "#todo", ":todo":
		return true
	}
	return false
}

func isBlockTextEmpty(block *notionapi.Block) bool {
	u.PanicIf(block.Type != notionapi.BlockText, "only supported on '%s' block, called on '%s' block", notionapi.BlockText, block.Type)
	blocks := block.InlineContent
	return len(blocks) == 0
}

func (c *Converter) isLastBlock() bool {
	lastIdx := len(c.converter.CurrBlocks) - 1
	return c.converter.CurrBlockIdx == lastIdx
}

func (c *Converter) isFirstBlock() bool {
	return c.converter.CurrBlockIdx == 0
}

// RenderText renders BlockText
func (c *Converter) RenderText(block *notionapi.Block) bool {
	sf := c.page.blockCodeToSourceFile[block.ID]
	if sf != nil {
		c.genSourceFile(sf)
		return true
	}

	if isBlockTextTodo(block) {
		return true
	}

	// notionapi/tohtml renders empty blocks as visible, so skip empty text
	// blocks if they are the first or last. Assumption is that it's careless
	// editing
	skipIfEmpty := c.isLastBlock() || c.isFirstBlock()
	if skipIfEmpty && isBlockTextEmpty(block) {
		return true
	}

	// TODO: convert to div
	c.converter.Printf(`<p>`)
	c.converter.RenderInlines(block.InlineContent)
	c.converter.RenderChildren(block)
	c.converter.Printf(`</p>`)
	return true
}

func (c *Converter) blockRenderOverride(block *notionapi.Block) bool {
	switch block.Type {
	case notionapi.BlockPage:
		return c.RenderPage(block)
	case notionapi.BlockCode:
		return c.RenderCode(block)
	case notionapi.BlockImage:
		return c.RenderImage(block)
	case notionapi.BlockText:
		return c.RenderText(block)
	case notionapi.BlockEmbed:
		return c.RenderEmbed(block)
	}
	return false
}

// Gen returns generated HTML
func (c *Converter) Gen() []byte {
	inner, err := c.converter.ToHTML()
	must(err)

	rootPage := c.page.NotionPage.Root()
	f := rootPage.FormatPage()
	isMono := f != nil && f.PageFont == "mono"

	s := ``
	if isMono {
		s += `<div style="font-family: monospace">`
	}
	s += string(inner)
	if isMono {
		s += `</div>`
	}
	return []byte(s)
}

func getInlinesPlain(a []*notionapi.TextSpan) string {
	s := ""
	for _, b := range a {
		s += b.Text
	}
	return s
}

func notionToHTML(page *Page, book *Book) []byte {
	// This is artificially generated page (e.g. contributors page)
	if page.NotionPage == nil {
		return []byte(page.BodyHTML)
	}

	logvf("Generating HTML for %s\n", page.NotionURL())
	res := Converter{
		book: book,
		page: page,
	}

	r := tohtml.NewConverter(page.NotionPage)
	notionapi.PanicOnFailures = true
	r.AddHeaderAnchor = true
	r.RenderBlockOverride = res.blockRenderOverride
	r.RewriteURL = res.rewriteURL
	res.converter = r

	html := res.Gen()

	return html
}
