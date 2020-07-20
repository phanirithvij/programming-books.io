package main

import (
	"encoding/json"
	"strings"

	"github.com/kjk/notionapi"
	"github.com/kjk/u"
)

/*
Generates a javascript file that looks like:

gTocItems = [
	[${chapter or aticle id}, ${parentIdx}, ${childIdx}, ${title}, ${synonym 1}, ${synonym 2}, ...],
];

It's saved in wwww/toc_search.js

Maybe: optimize this as a one long string to search instead of multiple strings
in an array. Use 0 to separate chapter/article. Use 1 to separate title
from synonims.

Also, have original and lower-cased version of the string. We search lower-cased
but show the original. That avoids lowercasing during search.
*/

const (
	itemIdxURL          = 0
	itemIdxParent       = 1
	itemIdxFirstChild   = 2
	itemIdxTitle        = 3
	itemIdxFirstSynonym = 4
)

func genTocItem(page *Page, idx int) []any {
	title := strings.TrimSpace(page.Title)
	uri := page.URLLastPath()
	return []interface{}{uri, idx, -1, title}
}

func genHeadings(page *Page, idx int) [][]any {
	var res [][]any

	uri := page.URLLastPath()
	headings := page.Headings
	for _, heading := range headings {
		title := heading.Text
		id := heading.ID
		if len(id) > 0 {
			id = uri + "#" + notionapi.ToDashID(id)
		}
		tocItem := []any{id, idx, -1, title}
		res = append(res, tocItem)
	}

	return res
}

// TODO: make it recursive and with arbitrary nesting
func genBookTOCSearchMust(book *Book) {
	var toc [][]any
	for _, chapter := range book.Chapters() {
		tocItem := genTocItem(chapter, -1)

		toc = append(toc, tocItem)
		chapIdx := len(toc) - 1
		u.PanicIf(chapIdx < 0)

		headings := genHeadings(chapter, chapIdx)
		toc = append(toc, headings...)

		for _, page := range chapter.Pages {
			tocItem = genTocItem(page, chapIdx)
			for _, syn := range page.getSearch() {
				tocItem = append(tocItem, syn)
			}
			toc = append(toc, tocItem)

			pageIdx := len(toc) - 1
			headings := genHeadings(page, pageIdx)
			toc = append(toc, headings...)
		}
	}

	// set first child idx from parent idx
	for i, tocItem := range toc {
		parentIdx := tocItem[itemIdxParent].(int)
		if parentIdx != -1 {
			parentTocItem := toc[parentIdx]
			idx := parentTocItem[itemIdxFirstChild].(int)
			if idx == -1 {
				parentTocItem[itemIdxFirstChild] = i
			}
		}
	}

	var d []byte
	var err error
	if doMinify {
		d, err = json.Marshal(toc)
	} else {
		d, err = json.MarshalIndent(toc, "", "  ")
	}
	u.PanicIfErr(err)
	s := "gTocItems = " + string(d) + ";"
	book.tocData = []byte(s)

	updateBookAppJS(book)
}
