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

// TODO: make it recursive and with arbitrary nesting
func genBookTOCSearchMust(book *Book) {
	var toc [][]interface{}
	for _, chapter := range book.Chapters() {
		title := strings.TrimSpace(chapter.Title)
		uri := chapter.URLLastPath()
		tocItem := []interface{}{uri, -1, -1, title}
		toc = append(toc, tocItem)
		chapIdx := len(toc) - 1
		u.PanicIf(chapIdx < 0)

		headings := chapter.Headings
		for _, heading := range headings {
			title := heading.Text
			id := heading.ID
			if len(id) > 0 {
				id = uri + "#" + notionapi.ToDashID(id)
			}
			tocItem = []interface{}{id, chapIdx, -1, title}
			toc = append(toc, tocItem)
		}

		for _, article := range chapter.Pages {
			title := strings.TrimSpace(article.Title)
			uri := article.URLLastPath()
			tocItem = []interface{}{uri, chapIdx, -1, title}
			for _, syn := range article.getSearch() {
				tocItem = append(tocItem, syn)
			}
			toc = append(toc, tocItem)

			headings := article.Headings
			articleIdx := len(toc) - 1
			for _, heading := range headings {
				title := heading.Text
				id := heading.ID
				if len(id) > 0 {
					id = uri + "#" + notionapi.ToDashID(id)
				}
				tocItem = []interface{}{id, articleIdx, -1, title}
				toc = append(toc, tocItem)
			}
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
