package main

import (
	"path/filepath"
	"sync/atomic"

	"github.com/kjk/notionapi"
)

// convert 2131b10c-ebf6-4938-a127-7089ff02dbe4 to 2131b10cebf64938a1277089ff02dbe4
// TODO: replace with direct use of notionapi.ToNoDashID
func toNoDashID(id string) string {
	return notionapi.ToNoDashID(id)
}

func downloadImage(d *notionapi.CachingClient, page *Page, block *notionapi.Block, link string) {
	rsp, err := d.DownloadFile(link, block)
	if err != nil {
		id := toNoDashID(page.NotionID)
		logf(ctx(), "downloadImage('%s') from page https://notion.so/%s failed with '%s'\n", link, id, err)
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

func downloadBook(book *Book, cachingPolicy notionapi.CachingPolicy) {
	book.cache = loadCache(book)
	createDirMust(book.NotionCacheDir)
	logf(ctx(), "Downloading %s, created cache dir: '%s'\n", book.Title, book.NotionCacheDir)

	c := &notionapi.Client{}
	//c.Logger = logFile
	//c.Logger = os.Stdout
	//c.DebugLog = true
	cacheDir := book.NotionCacheDir
	createDirMust(cacheDir)
	d, err := notionapi.NewCachingClient(cacheDir, c)
	must(err)
	d.CacheDirFiles = filepath.Join(cacheDir, "img")
	book.client = d

	d.PreLoadCache()
	startPageID := book.NotionStartPageID

	nDownloaded := 0
	nTotalPages := 0
	afterPageDownload := func(di *notionapi.DownloadInfo) error {
		nTotalPages++
		if di.FromCache {
			di.Page.NotionID = notionapi.NewNotionID(di.Page.ID)
			atomic.AddInt32(&nTotalDownloaded, 1)
			if nTotalPages == 1 || nTotalPages%64 == 0 {
				logf(ctx(), "CACHE '%s' %d\n", di.Page.NotionID.NoDashID, nTotalPages)
			}
		} else {
			atomic.AddInt32(&nTotalFromCache, 1)
			nDownloaded++
			logf(ctx(), "DL    '%s', %d\n", di.Page.NotionID.NoDashID, nTotalPages)
		}
		page := di.Page
		id := page.GetNotionID().NoDashID
		p := &Page{
			NotionPage: page,
			NotionID:   id,
			Book:       book,
		}
		book.idToPage[id] = p
		if !flgImportNotion {
			evalCodeSnippetsForPage(p)
		}
		downloadImages(d, book, p)
		calcPageHeadings(p)
		return nil
	}

	pages, err := d.DownloadPagesRecursively(startPageID, afterPageDownload)
	must(err)
	nPages := len(pages)
	logf(ctx(), "Got %d pages for %s, downloaded: %d, from cache: %d\n", nPages, book.Title, d.DownloadedCount, d.FromCacheCount)
}
