package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/notionapi"
)

type any = interface{}

var (
	doMinify bool

	notionAuthToken string

	// when downloading pages from the server, count total number of
	// downloaded and those from cache
	nTotalDownloaded int32
	nTotalFromCache  int32
)

func isPreview() bool {
	return flgPreview
}

func initBook(book *Book) {
	book.DirOnDisk = filepath.Join("essential", book.DirShort)
	book.DirCache = filepath.Join("books", book.DirShort, "cache")
	book.NotionCacheDir = filepath.Join(book.DirCache, "notion")
	book.idToPage = map[string]*Page{}
}

var (
	flgPreview bool
	// disables downloading pages
	flgNoDownload     bool
	flgGistRedownload bool
	// will only download (no eval, no generation)
	flgDownloadOnly bool

	// disables notion cache, forcing re-download of notion page
	// even if cached verison on disk exits
	flgDisableNotionCache bool
)

func main() {
	var (
		flgGen          bool
		flgGenRender    bool
		flgBook         string
		flgDownloadGist string
		flgPreviewInsta bool
	)

	{
		flag.BoolVar(&flgPreview, "preview", false, "preview the book locally")
		flag.BoolVar(&flgGen, "gen", false, "generate a book and deploy preview")
		flag.BoolVar(&flgGenRender, "gen-render", false, "generate to a www_generated directory for deploying on render.com")
		flag.StringVar(&flgBook, "book", "", "name of the book")
		flag.BoolVar(&flgDownloadOnly, "download-only", false, "only download the books from notion (no eval, no html generation")
		flag.StringVar(&flgDownloadGist, "download-gist", "", "id of the gist to (re)download. Must also provide a book")
		flag.BoolVar(&flgDisableNotionCache, "no-cache", false, "if true, disables cache for notion")
		flag.BoolVar(&flgPreviewInsta, "preview-insta", false, "preview to instantpreview.dev")
		flag.Parse()
	}

	closeLog := openLog()
	defer closeLog()

	timeStart := time.Now()
	defer func() {
		logf(ctx(), "Downloaded %d pages, %d from cache. Total time: %s\n", nTotalDownloaded, nTotalFromCache, time.Since(timeStart))
	}()

	// ad-hoc, rarely done tasks
	if false {
		genTwitterImagesAndExit()
		return
	}
	if false {
		genSmallCoversAndExit()
		return
	}
	if false {
		optimizeAllImages()
		return
	}

	{
		//notionAuthToken = os.Getenv("NOTION_TOKEN")
		// we don't need authentication and the result change
		// in authenticated vs. non-authenticated state
		notionAuthToken = ""
		if notionAuthToken != "" {
			logf(ctx(), "NOTION_TOKEN provided, can write back\n")
		} else {
			logf(ctx(), "NOTION_TOKEN not provided, read only\n")
		}
	}

	notionapi.LogFunc = logsf
	notionapi.PanicOnFailures = true

	if flgDownloadGist != "" {
		book := findBook(flgBook)
		if book == nil {
			logf(ctx(), "-download-gist also requires valid -book, given: '%s'\n", flgBook)
		}
		downloadSingleGist(book, flgDownloadGist)
		return
	}

	booksToProcess := booksMain
	if flgBook != "" {
		bookNames := strings.Split(flgBook, ",")
		booksToProcess = nil
		for _, bookName := range bookNames {
			book := findBook(bookName)
			panicIf(book == nil, "'%s' is not a valid book name", flgBook)
			booksToProcess = append(booksToProcess, book)
		}
	}
	for _, book := range booksToProcess {
		initBook(book)
	}

	if flgPreview {
		previewWebsite(booksToProcess)
		return
	}

	if flgPreviewInsta {
		previewToInsantPreview(booksToProcess)
		return
	}

	if flgGen || flgGenRender {
		dir, _ := filepath.Abs("www_generated")
		os.RemoveAll(dir)
		genToDir(booksToProcess, dir)
		return
	}

	if flgDownloadOnly {
		logf(ctx(), "Downloading %d books\n", len(booksToProcess))
		n := len(booksToProcess)
		for i, book := range booksToProcess {
			downloadBook(book)
			logvf("downloaded book %d out of %d, name: %s, dir: %s\n", i+1, n, book.Title, book.DirShort)
		}
		return
	}

	flag.Usage()
}

// download a single gist and store in the cache for a given book
func downloadSingleGist(book *Book, gistID string) {
	bookName := book.DirShort
	logf(ctx(), "Downloading gist '%s' and storing in the cache for the book '%s'\n", gistID, bookName)
	cache := loadCache(book)
	gist := gistDownloadMust(gistID)
	didChange := cache.saveGist(gistID, gist.Raw)
	if didChange {
		logf(ctx(), "Saved a new or updated version of gist\n")
		return
	}
	logf(ctx(), "Gist didn't change!\n")
}
