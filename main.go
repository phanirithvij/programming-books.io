package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/notionapi"
	"github.com/kjk/u"
)

type any = interface{}

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
	case *notionapi.EventError:
		logf(v.Error)
	case *notionapi.EventDidDownload:
		nProcessed++
		nDownloadedPages++
		logf("%03d '%s' : downloaded in %s\n", nProcessed, v.PageID, v.Duration)
	case *notionapi.EventDidReadFromCache:
		nProcessed++
		nNotionPagesFromCache++
		// TODO: only verbose
		//logf("%03d '%s' : read from cache in %s\n", nProcessed, v.PageID, v.Duration)
	case *notionapi.EventGotVersions:
		logf("downloaded info about %d versions in %s\n", v.Count, v.Duration)
	}
}

func shouldCopyImage(path string) bool {
	return !strings.Contains(path, "@2x")
}

func copyCoversMust(dir string) {
	srcDir := "covers"
	dstDir := filepath.Join(dir, "covers")
	u.DirCopyRecurMust(dstDir, srcDir, shouldCopyImage)
	dstDir = filepath.Join(dir, "covers_small")
	srcDir = filepath.Join("covers_small")
	u.DirCopyRecurMust(dstDir, srcDir, shouldCopyImage)
}

func copyImages(book *Book) {
	src := filepath.Join(book.NotionCacheDir, "img")
	if !u.DirExists(src) {
		return
	}
	dst := filepath.Join(book.destDir(), "img")
	u.DirCopyRecurMust(dst, src, nil)
}

func isPreview() bool {
	return flgPreview
}

var (
	flgAnalytics                bool
	flgPreview                  bool
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

	gDestDir string
)

func main() {
	var (
		flgGen          bool
		flgBook         string
		flgAllBooks     bool
		flgWc           bool
		flgDownloadGist string
	)

	{
		dir := filepath.Join("..", "generated")
		dir, err := filepath.Abs(dir)
		must(err)
		gDestDir = dir
		indexDestDir = filepath.Join(gDestDir, "www")
	}

	{
		flag.BoolVar(&flgAnalytics, "analytics", false, "add google analytics code")
		flag.BoolVar(&flgWc, "wc", false, "wc -l")
		flag.BoolVar(&flgDeployProd, "deploy-prod", false, "deploy to prodution")
		flag.BoolVar(&flgPreview, "preview", false, "if true, runs vercel dev to preview the book")
		flag.BoolVar(&flgDeployDev, "deploy-dev", false, "deploy to dev")
		flag.BoolVar(&flgClean, "clean", false, "if true, re-create 'www' directory")
		flag.BoolVar(&flgGen, "gen", false, "generate a book and deploy preview")
		flag.StringVar(&flgBook, "book", "", "name of the book")
		flag.BoolVar(&flgAllBooks, "all-books", false, "if true, apply to all books")
		flag.StringVar(&flgDownloadGist, "download-gist", "", "id of the gist to (re)download. Must also provide a book")
		flag.BoolVar(&flgDisableNotionCache, "no-cache", false, "if true, disables cache for notion")
		flag.Parse()

		// change to true for easier ad-hoc debugging in visual studio code
		if false {
			//flgBook = "go"
			flgAllBooks = true
			flgGen = true
		}

		if flgPreview {
			flgGen = true
		}

		if flgDeployProd {
			flgAnalytics = true
			flgGen = true
		}

		if flgAllBooks {
			flgClean = true
			panicIf(flgPreview, "-preview is not compatible with -all-books")
		}

		// if flgDeployDev || flgDeployProd {
		// 	panicIf(flgPreview, "-preview is not compatible with -deploy-prod or -deploy-dev")
		// }

		if flgAnalytics {
			googleAnalyticsTmpl := `
			<script async src="https://www.googletagmanager.com/gtag/js?id=UA-113489735-1"></script>
			<script>
				window.dataLayer = window.dataLayer || [];
				function gtag(){dataLayer.push(arguments);}
				gtag('js', new Date());

				gtag('config', 'UA-113489735-1');
			</script>
		`
			googleAnalytics = template.HTML(googleAnalyticsTmpl)
		}

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

	// ad-hoc, rarely done tasks
	if false {
		//genTwitterImagesAndExit()
		//optimizeAllImages()
		return
	}

	if flgWc {
		doLineCount()
		return
	}

	if flgGen {
		updateGeneratedRepo()
		buildFrontend()

		if flgBook == "index" {
			genBookIndexAndDeploy(allBooks)
			//commitAndPushGeneratedRepo
			return
		}
		if flgAllBooks {
			genBookIndexAndDeploy(allBooks)
			n := len(allBooks)
			for i, book := range allBooks {
				book = findBook(book.DirShort)
				generateBookAndDeploy(book)
				fmt.Printf("book %d out of %d, name: %s, dir: %s\n", i+1, n, book.Title, book.DirShort)
			}
			//commitAndPushGeneratedRepo
			return
		}
		book := findBook(flgBook)
		generateBookAndDeploy(book)
		fmt.Printf("book: %s, dir: %s\n", book.Title, book.DirShort)
		if flgPreview {
			//previewBook(book)
		}
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

func previewBook(book *Book) {
	panic("previewBook NYI")
}

func updateGeneratedRepo() {
	dir := gDestDir
	if !u.PathExists(dir) {
		fmt.Printf("updateGeneratedRepo: directory %s doesn't exist. Must git clone https://github.com/essentialbooks/generated there\n", dir)
		os.Exit(1)
	}
	u.EnsureGitClean(dir)
	u.GitPullMust(dir)
}

func commitAndPushGeneratedRepo() {
	dir := gDestDir
	{
		cmd := exec.Command("git", "add", "www")
		cmd.Dir = dir
		u.RunCmdMust(cmd)
	}
	{
		cmd := exec.Command("git", "commit", "-am", "update generated html")
		cmd.Dir = dir
		u.RunCmdMust(cmd)
	}
	{
		cmd := exec.Command("git", "push")
		cmd.Dir = dir
		u.RunCmdMust(cmd)
	}
}
