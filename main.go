package main

import (
	"flag"
	"fmt"
	"html/template"
	"os/exec"
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
	flgPreviewStatic            bool
	flgPreviewOnDemand          bool
	flgReportStackOverflowLinks bool
	// if true, disables downloading pages
	flgNoDownload     bool
	flgGistRedownload bool
	flgDeployProd     bool
	flgDeployDev      bool

	// if true, re-create "www" directory
	flgClean bool
	// if true, disables notion cache, forcing re-download of notion page
	// even if cached verison on disk exits
	flgDisableNotionCache bool
)

func main() {
	var (
		flgGen          bool
		flgBook         string
		flgWc           bool
		flgDownloadGist string
	)

	{
		flag.BoolVar(&flgWc, "wc", false, "wc -l")
		flag.BoolVar(&flgDeployProd, "deploy-prod", false, "deploy to prodution")
		flag.BoolVar(&flgDeployDev, "deploy-dev", false, "deploy to dev")
		flag.BoolVar(&flgClean, "clean", false, "if true, re-create 'www' directory")
		flag.BoolVar(&flgGen, "gen", false, "generate a book and deploy preview")
		flag.StringVar(&flgBook, "book", "", "name of the book")
		flag.StringVar(&flgDownloadGist, "download-gist", "", "id of the gist to (re)download. Must also provide a book")
		flag.BoolVar(&flgDisableNotionCache, "no-cache", false, "if true, disables cache for notion")
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

	if flgWc {
		doLineCount()
		return
	}

	if flgGen {
		book := findBook(flgBook)
		generateBookAndDeploy(book)
		fmt.Printf("book: %s, dir: %s\n", book.Title, book.DirShort)
		return
	}

	if flgDownloadGist != "" {
		book := findBook(flgBook)
		downloadSingleGist(book, flgDownloadGist)
		return
	}

	flag.Usage()
}

func generateBookAndDeploy(book *Book) {
	downloadBook(book)

	buildFrontend()

	genBook(book)
	deployWithVercel(book)
}

func newNotionClient() *notionapi.Client {
	client := &notionapi.Client{
		AuthToken: notionAuthToken,
	}
	// client.Logger = logFile
	return client
}

// download a single gist and store in the cache for a given book
func downloadSingleGist(book *Book, gistID string) {
	bookName := book.DirShort
	logf("Downloading gist '%s' and storing in the cache for the book '%s'\n", gistID, bookName)
	cache := loadCache(book)
	gist := gistDownloadMust(gistID)
	didChange := cache.saveGist(gistID, gist.Raw)
	if didChange {
		logf("Saved a new or updated version of gist\n")
		return
	}
	logf("Gist didn't change!\n")
}

// vercel www --scope teamkjk --name book-git --confirm
func deployWithVercel(book *Book) {
	if !(flgDeployProd || flgDeployDev) {
		// no deploying
		fmt.Printf("not deploying\n")
		return
	}
	projectName := "book-" + book.DirShort
	args := []string{"www", "--scope", "teamkjk", "--name", projectName, "--confirm"}
	if flgDeployProd {
		args = append(args, "--prod")
	}
	cmd := exec.Command("vercel", args...)

	cmd.Dir = book.DirOnDisk
	u.RunCmdLoggedMust(cmd)
}
