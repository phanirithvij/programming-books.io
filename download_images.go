package main

import (
	"path/filepath"

	"github.com/kjk/notionapi"
)

func downloadImage(d *notionapi.CachingClient, page *Page, block *notionapi.Block, link string) {
	rsp, err := d.DownloadFile(link, block)
	if err != nil {
		id := toNoDashID(page.NotionID)
		logf("downloadImage('%s') from page https://notion.so/%s failed with '%s'\n", link, id, err)
		must(err)
	}
	path := rsp.CacheFilePath
	relURL := "/essential/" + page.Book.DirShort + "/img/" + filepath.Base(path)
	im := &ImageMapping{
		link:        link,
		path:        path,
		relativeURL: relURL,
	}
	page.Images = append(page.Images, im)
}

func downloadImages(d *notionapi.CachingClient, book *Book, page *Page) {
	handleImage := func(block *notionapi.Block) {
		downloadImage(d, page, block, block.Source)
	}

	fn := func(block *notionapi.Block) {
		switch block.Type {
		case notionapi.BlockImage:
			handleImage(block)
		}
	}
	root := page.NotionPage.Root()
	format := root.FormatPage()
	if format.PageCover != "" {
		downloadImage(d, page, root, format.PageCover)
	}
	page.NotionPage.ForEachBlock(fn)
}
