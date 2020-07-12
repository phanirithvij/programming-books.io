package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/notionapi"
	"github.com/kjk/notionapi/caching_downloader"
	"github.com/kjk/u"
)

var (
	googleAnalytics template.HTML
	doMinify        bool

	notionAuthToken string

	// when downloading pages from the server, count total number of
	// downloaded and those from cache
	nTotalDownloaded int
	nTotalFromCache  int
)

var (
	nProcessed            = 0
	nNotionPagesFromCache = 0
	nDownloadedPages      = 0
)

func eventObserver(ev interface{}) {
	switch v := ev.(type) {
	case *caching_downloader.EventError:
		logf(v.Error)
	case *caching_downloader.EventDidDownload:
		nProcessed++
		nDownloadedPages++
		logf("%03d '%s' : downloaded in %s\n", nProcessed, v.PageID, v.Duration)
	case *caching_downloader.EventDidReadFromCache:
		nProcessed++
		nNotionPagesFromCache++
		// TODO: only verbose
		//logf("%03d '%s' : read from cache in %s\n", nProcessed, v.PageID, v.Duration)
	case *caching_downloader.EventGotVersions:
		logf("downloaded info about %d versions in %s\n", v.Count, v.Duration)
	}
}

func shouldCopyImage(path string) bool {
	return !strings.Contains(path, "@2x")
}

func copyCoversMust() {
	srcDir := "covers"
	dstDir := filepath.Join("www", "covers")
	u.DirCopyRecurMust(dstDir, srcDir, shouldCopyImage)
	dstDir = filepath.Join("www", "covers_small")
	srcDir = filepath.Join("covers", "covers_small")
	u.DirCopyRecurMust(dstDir, srcDir, shouldCopyImage)
}

func copyImages(book *Book) {
	src := filepath.Join(book.NotionCacheDir(), "img")
	if !u.DirExists(src) {
		return
	}
	dst := filepath.Join(book.destDir(), "img")
	u.DirCopyRecurMust(dst, src, nil)
}

func isPreview() bool {
	return flgPreviewStatic || flgPreviewOnDemand
}

var (
	// url or id of the page to rebuild
	flgNoUpdateOutput bool
	// if true, disables notion cache, forcing re-download of notion page
	// even if cached verison on disk exits
	flgDisableNotionCache       bool
	flgPreviewStatic            bool
	flgPreviewOnDemand          bool
	flgReportStackOverflowLinks bool
	// if true, disables downloading pages
	flgNoDownload     bool
	flgGistRedownload bool
)

func main() {
	var (
		flgBook string
		flgWc   bool
	)

	{
		flag.BoolVar(&flgWc, "wc", false, "wc -l")
		flag.StringVar(&flgBook, "book", "go", "book to generate")
		flag.Parse()
	}

	closeLog := openLog()
	defer closeLog()

	timeStart := time.Now()
	defer func() {
		logf("Downloaded %d pages, %d from cache. Total time: %s\n", nTotalDownloaded, nTotalFromCache, time.Since(timeStart))
	}()

	{
		//notionAuthToken = os.Getenv("NOTION_TOKEN")
		// we don't need authentication and the result change
		// in authenticated vs. non-authenticated state
		notionAuthToken = ""
		if notionAuthToken != "" {
			logf("NOTION_TOKEN provided, can write back\n")
		} else {
			logf("NOTION_TOKEN not provided, read only\n")
		}
	}

	notionapi.LogFunc = logf

	os.RemoveAll("www")
	defer func() {
		os.RemoveAll("www")
	}()
	buildFrontend()

	book := findBook(flgBook)
	generateBook(book)
	fmt.Printf("book: %s, dir: %s\n", book.Title, book.DirShort)
}

func generateBook(book *Book) {
	initBook(book)
	downloadBook(book)
	currBookDir = book.DirOnDisk
	genBook(book)
}

func newNotionClient() *notionapi.Client {
	client := &notionapi.Client{
		AuthToken: notionAuthToken,
	}
	// client.Logger = logFile
	return client
}

// download a single gist and store in the cache for a given book
func downloadSingleGist(gistID string) {
	// must have 1 remaining arg that is book name
	restArgs := flag.Args()
	if len(restArgs) != 0 {
		logf("-download-gist expects a name of a single book to use for cache\n")
		logf("remaining args are: '%#v'\n", restArgs)
	}
	bookName := restArgs[0]
	logf("Downloading gist '%s' and storing in the cache for the book '%s'\n", gistID, bookName)
	path := filepath.Join("cache", bookName, "cache.txt")
	cache := loadCache(path)
	gist := gistDownloadMust(gistID)
	cache.saveGist(gistID, gist.Raw)
	logf("Saved a gist\n")
}
