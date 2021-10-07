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

const (
	dirWwwGenerated = "www_generated"
	httpPort        = 9044
)

var (
	doMinify bool

	// when downloading pages from the server, count total number of
	// downloaded and those from cache
	nTotalDownloaded int32
	nTotalFromCache  int32
)

func isPreview() bool {
	return flgRunServer
}

func initBook(book *Book) {
	book.DirOnDisk = filepath.Join("essential", book.DirShort)
	book.DirCache = filepath.Join("books", book.DirShort, "cache")
	book.NotionCacheDir = filepath.Join(book.DirCache, "notion")
	book.idToPage = map[string]*Page{}
}

var (
	flgRunServer bool
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
		flgGen             bool
		flgBook            string
		flgDownloadGist    string
		flgRunServerStatic bool
	)

	{
		flag.BoolVar(&flgRunServer, "run", false, "run dev server")
		flag.BoolVar(&flgRunServerStatic, "run-static", false, "run prod server serving www_generated")
		flag.BoolVar(&flgGen, "gen", false, "generate a book and deploy preview")
		flag.StringVar(&flgBook, "book", "", "name of the book")
		flag.BoolVar(&flgDownloadOnly, "download-only", false, "only download the books from notion (no eval, no html generation")
		flag.StringVar(&flgDownloadGist, "download-gist", "", "id of the gist to (re)download. Must also provide a book")
		flag.BoolVar(&flgDisableNotionCache, "no-cache", false, "if true, disables cache for notion")
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

	booksToProcess := getAllBooks()
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

	if flgRunServer {
		runServerDynamic(booksToProcess)
		return
	}

	if flgRunServerStatic {
		runServerStatic()
		return
	}

	if flgGen {
		dir, _ := filepath.Abs(dirWwwGenerated)
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
