package main

import (
	"flag"
	"os"
	"path/filepath"
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
	flgNoCleanCheck       bool

	gDestDir string
)

func main() {
	var (
		flgGen          bool
		flgGenRender    bool
		flgBook         string
		flgAllBooks     bool
		flgDownloadGist string
		flgCheckinHTML  bool
		flgRebuildAll   bool
		flgPreviewInsta bool
	)

	{
		dir := filepath.Join("..", "generated")
		dir, err := filepath.Abs(dir)
		must(err)
		gDestDir = dir
		indexDestDir = filepath.Join(gDestDir, "www")
	}

	{
		flag.BoolVar(&flgNoCleanCheck, "no-clean-check", false, "don't check if destination directory is not clean")
		flag.BoolVar(&flgPreview, "preview", false, "preview the book locally")
		flag.BoolVar(&flgGen, "gen", false, "generate a book and deploy preview")
		flag.BoolVar(&flgGenRender, "gen-render", false, "generate to a www_generated directory for deploying on render.com")
		flag.StringVar(&flgBook, "book", "", "name of the book")
		flag.BoolVar(&flgAllBooks, "all-books", false, "if true, apply to all books")
		flag.BoolVar(&flgDownloadOnly, "download-only", false, "only download the books from notion (no eval, no html generation")
		flag.StringVar(&flgDownloadGist, "download-gist", "", "id of the gist to (re)download. Must also provide a book")
		flag.BoolVar(&flgCheckinHTML, "checkin-html", false, "checkin generated html")
		flag.BoolVar(&flgDisableNotionCache, "no-cache", false, "if true, disables cache for notion")
		flag.BoolVar(&flgRebuildAll, "rebuild-all", false, "same as -books-all -clean -gen")
		flag.BoolVar(&flgPreviewInsta, "preview-insta", false, "preview to instantpreview.dev")
		flag.Parse()

		// change to true for easier ad-hoc debugging in visual studio code
		if false {
			//flgBook = "go"
			flgAllBooks = true
			flgGen = true
		}

		if flgRebuildAll {
			flgAllBooks = true
			flgGen = true
		}
	}

	closeLog := openLog()
	defer closeLog()

	timeStart := time.Now()
	defer func() {
		logf(ctx(), "Downloaded %d pages, %d from cache. Total time: %s\n", nTotalDownloaded, nTotalFromCache, time.Since(timeStart))
	}()

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

	if flgDownloadGist != "" {
		book := findBook(flgBook)
		if book == nil {
			logf(ctx(), "-download-gist also requires valid -book, given: '%s'\n", flgBook)
		}
		initBook(book)
		downloadSingleGist(book, flgDownloadGist)
		return
	}

	if flgPreview {
		previewWebsite(allBooks)
		return
	}

	if flgPreviewInsta {
		previewToInsantPreview(allBooks)
		return
	}

	var booksToProcess []*Book
	if flgBook != "" {
		book := findBook(flgBook)
		panicIf(book == nil, "'%s' is not a valid book name", flgBook)
		booksToProcess = []*Book{book}
	}
	if flgAllBooks {
		booksToProcess = allBooks
	}

	if flgGen {
		genToDir(allBooks, indexDestDir)
		return
	}

	if flgGenRender {
		//dir := filepath.Join("..", "progbooks_generated")
		dir := "www_generated"
		dir, _ = filepath.Abs(dir)
		os.RemoveAll(dir)
		genToDir(allBooks, dir)
		return
	}

	for _, book := range booksToProcess {
		initBook(book)
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
