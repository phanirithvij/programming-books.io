package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kjk/common/httputil"
	"github.com/kjk/common/server"
	"github.com/kjk/notionapi"
)

func genIndexGrid(books []*Book, w io.Writer) error {
	panicIf(w == nil)
	d := struct {
		PageCommon
		Books []*Book
	}{
		PageCommon: getPageCommon(),
		Books:      books,
	}
	return execTemplateToWriter("index-grid.tmpl.html", d, w)
}

func genFeedback(w io.Writer) error {
	d := struct {
		PageCommon
	}{
		PageCommon: getPageCommon(),
	}
	return execTemplateToWriter("feedback.tmpl.html", d, w)
}

func genAbout(w io.Writer) error {
	d := getPageCommon()
	return execTemplateToWriter("about.tmpl.html", d, w)
}

// generates book for www.programming-books.io
func genIndex(books []*Book, w io.Writer) error {
	leftBooks, rightBooks := splitBooks(books)
	d := struct {
		PageCommon
		Books      []*Book
		LeftBooks  []*Book
		RightBooks []*Book
		NotionURL  string
	}{
		PageCommon: getPageCommon(),
		Books:      books,
		LeftBooks:  leftBooks,
		RightBooks: rightBooks,
		NotionURL:  gitHubBaseURL,
	}

	return execTemplateToWriter("index.tmpl.html", d, w)
}

func gen404Indexl(w io.Writer) error {
	d := struct {
		PageCommon
		Book *Book
	}{
		PageCommon: getPageCommon(),
	}
	return execTemplateToWriter("404-index.tmpl.html", d, w)
}

func isFullURL(uri string) bool {
	return strings.HasPrefix(uri, "https://") || strings.HasPrefix(uri, "http://")
}

const (
	sitemapTmpl = `User-agent: *
Disallow:

Sitemap: %s
`
)

func genSitemapTxt(books []*Book) []byte {
	// http://www.advancedhtml.co.uk/robots-sitemaps.htm

	var urls []string

	addSitemapURL := func(b *Book, uri string) {
		if !isFullURL(uri) {
			uri = urlJoin(b.CanonnicalURL(), uri)
		}
		urls = append(urls, uri)
	}

	for _, book := range books {
		addSitemapURL(book, "/")
		for _, chapter := range book.Chapters() {
			addSitemapURL(book, chapter.URL())
			for _, article := range chapter.Pages {
				addSitemapURL(book, article.URL())
			}
		}
		//addSitemapURL(b, "/about")
	}
	sort.Strings(urls)
	s := strings.Join(urls, "\n")
	return []byte(s)
}

func genRobotsTxt() []byte {
	sitemapURL := urlJoin(siteBaseURL, "sitemap.txt")
	robotsTxt := fmt.Sprintf(sitemapTmpl, sitemapURL)
	return []byte(robotsTxt)
}

func serveStart(w http.ResponseWriter, r *http.Request, uri string) {
	if r == nil {
		return
	}
	ct := mimeTypeFromFileName(uri)
	w.Header().Add("Content-Type", ct)
	w.WriteHeader(http.StatusOK) // 200
}

func makeServerGet(books []*Book) server.GetHandlerFunc {
	return func(uri string) func(w http.ResponseWriter, r *http.Request) {
		//logf(ctx(), "serverGet: '%s'\n", uri)
		switch uri {
		case "/index.html":
			return func(w http.ResponseWriter, r *http.Request) {
				//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
				serveStart(w, r, uri)
				genIndex(books, w)
			}
		case "/index-grid.html":
			return func(w http.ResponseWriter, r *http.Request) {
				//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
				serveStart(w, r, uri)
				genIndexGrid(books, w)
			}
		case "/404.html":
			return func(w http.ResponseWriter, r *http.Request) {
				//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
				serveStart(w, r, uri)
				gen404Indexl(w)
			}
		case "/about.html":
			return func(w http.ResponseWriter, r *http.Request) {
				//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
				serveStart(w, r, uri)
				genAbout(w)
			}
		case "/feedback.html":
			return func(w http.ResponseWriter, r *http.Request) {
				//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
				serveStart(w, r, uri)
				genFeedback(w)
			}
		case "/robots.txt":
			return func(w http.ResponseWriter, r *http.Request) {
				//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
				serveStart(w, r, uri)
				d := genRobotsTxt()
				w.Write(d)
			}
		case "/sitemap.txt":
			return func(w http.ResponseWriter, r *http.Request) {
				//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
				serveStart(w, r, uri)
				d := genSitemapTxt(books)
				w.Write(d)
			}
		}
		return nil
	}
}

func genBooksIndexHandler(books []*Book) server.Handler {
	serverURLS := func() []string {
		files := []string{
			"/index.html",
			"/index-grid.html",
			"/404.html",
			"/about.html",
			"/feedback.html",
			"/robots.txt",
			"/sitemap.txt",
		}
		return files
	}
	return server.NewDynamicHandler(makeServerGet(books), serverURLS)
}

var (
	rxExtractPageID = regexp.MustCompile(`(?m)[0-9a-f]{32}`)
)

func makeHTTPServer(srv *server.Server) *http.Server {
	panicIf(srv == nil, "must provide srv")
	httpPort := 8080
	if srv.Port != 0 {
		httpPort = srv.Port
	}
	httpAddr := fmt.Sprintf(":%d", httpPort)
	if isWindows() {
		httpAddr = "localhost" + httpAddr
	}

	mainHandler := func(w http.ResponseWriter, r *http.Request) {
		//logf(ctx(), "mainHandler: '%s'\n", r.RequestURI)
		timeStart := time.Now()
		cw := httputil.CapturingResponseWriter{ResponseWriter: w}

		defer func() {
			if p := recover(); p != nil {
				logf(ctx(), "mainHandler: panicked with with %v\n", p)
				http.Error(&cw, fmt.Sprintf("Error: %v", p), http.StatusInternalServerError)
			}
			logHTTPReq(r, cw.StatusCode, cw.Size, time.Since(timeStart))
		}()

		uri := r.URL.Path
		serve, is404 := srv.FindHandler(uri)
		if is404 {
			// try to handle /essential/go/4c4df97de2e241dabade237cefe4c6d4-variables-of-function-type
			// by redirecting to the actual URL
			findPageIDMatch := func() string {
				matches := rxExtractPageID.FindAllString(uri, 1)
				if len(matches) == 1 {
					pageID := matches[0]
					for _, h := range srv.Handlers {
						for _, uri := range h.URLS() {
							if strings.Contains(uri, pageID) {
								return uri
							}
						}
					}
				}
				return ""
			}
			redirectURL := findPageIDMatch()
			if redirectURL != "" {
				http.Redirect(&cw, r, redirectURL, http.StatusTemporaryRedirect) // 307
				return
			}
		}

		if serve != nil {
			serve(&cw, r)
			return
		}

		http.NotFound(&cw, r)
	}

	httpSrv := &http.Server{
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second, // introduced in Go 1.8
		Handler:      http.HandlerFunc(mainHandler),
	}
	httpSrv.Addr = httpAddr
	return httpSrv
}

func runServerProd() {
	panicIf(!dirExists(dirWwwGenerated))
	h := server.NewDirHandler(dirWwwGenerated, "/", nil)
	logf(ctx(), "runServerProd starting, hasSpacesCreds: %v, %d urls\n", hasSpacesCreds(), len(h.URLS()))
	srv := &server.Server{
		Handlers:  []server.Handler{h},
		CleanURLS: true,
		Port:      httpPort,
	}

	closeHTTPLog := OpenHTTPLog("progbooks")
	defer closeHTTPLog()

	httpSrv := makeHTTPServer(srv)
	logf(ctx(), "Starting server on http://%s'\n", httpSrv.Addr)
	if isWindows() {
		openBrowser(fmt.Sprintf("http://%s", httpSrv.Addr))
	}
	err := httpSrv.ListenAndServe()
	logf(ctx(), "runServerProd: httpSrv.ListenAndServe() returned '%s'\n", err)
}

func runServerDynamic(booksToProcess []*Book) {
	logf(ctx(), "runServerDynamic: start for %d books\n", len(booksToProcess))
	timeStart := time.Now()
	flgReloadTemplates = true
	srv := buildServer(booksToProcess, true)
	go func() {
		waitBuildServerDone()
		// TODO: mutex protection
		nPages := 0
		for _, h := range srv.Handlers {
			nPages += len(h.URLS())
		}
		logf(ctx(), "runServerDynamic: finished %d urls in %s\n", nPages, time.Since(timeStart))
	}()

	closeHTTPLog := OpenHTTPLog("progbooks")
	defer closeHTTPLog()

	httpSrv := makeHTTPServer(srv)
	logf(ctx(), "Starting server on http://%s'\n", httpSrv.Addr)
	if isWindows() {
		openBrowser(fmt.Sprintf("http://%s", httpSrv.Addr))
	}
	err := httpSrv.ListenAndServe()
	logf(ctx(), "runServerDynamic: httpSrv.ListenAndServe() returned '%s'\n", err)
}

func genToDir(booksToProcess []*Book, dir string) {
	logf(ctx(), "genToDir: '%s'\n", dir)
	must(os.RemoveAll(dir))

	timeStart := time.Now()
	flgReloadTemplates = false
	srv := buildServer(booksToProcess, false)
	waitBuildServerDone()
	nPages := 0
	for _, h := range srv.Handlers {
		nPages += len(h.URLS())
	}
	logf(ctx(), "genToDir: finished %d urls in %s\n", nPages, time.Since(timeStart))

	nFiles := 0
	totalSize := int64(0)
	onWritten := func(path string, d []byte) {
		fsize := int64(len(d))
		totalSize += fsize
		sizeStr := formatSize(fsize)
		if nFiles%256 == 0 {
			logf(ctx(), "genToDir: file %d '%s' of size %s\n", nFiles+1, path, sizeStr)
		}
		nFiles++
	}
	server.WriteServerFilesToDir(dir, srv.Handlers, onWritten)
}

var booksSem chan bool
var booksWg sync.WaitGroup

func waitBuildServerDone() {
	booksWg.Wait()
}

// call before genBookHandler
func initBookHandlers() {
	nThreads := runtime.NumCPU()
	logf(ctx(), "initBookHandler: %d threads\n", nThreads)
	booksSem = make(chan bool, nThreads)
}

func maybeRedownloadPage(book *Book, pageID string) *Page {
	if !isPreview() {
		return nil
	}
	logf(ctx(), "maybeRedownloadPage %s\n", pageID)

	c := &notionapi.Client{}
	//c.Logger = logFile
	c.Logger = os.Stdout
	//c.DebugLog = true
	cacheDir := book.NotionCacheDir
	cc, err := notionapi.NewCachingClient(cacheDir, c)
	must(err)
	cc.CacheDirFiles = filepath.Join(cacheDir, "img")
	cc.Policy = notionapi.PolicyDownloadAlways
	book.client = cc
	page, err := cc.DownloadPage(pageID)
	if err != nil {
		return nil
	}
	id := page.GetNotionID().NoDashID
	p := &Page{
		NotionPage: page,
		NotionID:   id,
		Book:       book,
	}
	evalCodeSnippetsForPage(p)
	downloadImages(cc, book, p)
	calcPageHeadings(p)
	return p
}

// returns immediately but builds the handler in the background
func genBookHandler(book *Book) server.Handler {
	panicIf(book == nil)
	var handlers []server.Handler
	var filesHandler *server.FilesHandler
	var mu sync.Mutex
	pages := map[string]*Page{} // maps url to Page

	addHandler := func(h server.Handler) {
		mu.Lock()
		handlers = append(handlers, h)
		mu.Unlock()
	}
	filesHandler = server.NewFilesHandler()
	addHandler(filesHandler)

	baseURL := book.URL()
	indexURL := path.Join(baseURL, "index.html")
	book404URL := path.Join(baseURL, "404.html")
	overviewURL := path.Join(baseURL, "overview.html")

	getURLs := func() []string {
		mu.Lock()
		defer mu.Unlock()
		urls := []string{
			indexURL, book404URL, overviewURL,
		}
		for uri := range pages {
			urls = append(urls, uri)
		}
		for _, h := range handlers {
			urls = append(urls, h.URLS()...)
		}
		return urls
	}

	get := func(uri string) func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch uri {
		case indexURL:
			return func(w http.ResponseWriter, r *http.Request) {
				genBookIndexHTML(book, w)
			}
		case book404URL:
			return func(w http.ResponseWriter, r *http.Request) {
				genBook404(book, w)
			}
		case overviewURL:
			d := genOverviewContent(book)
			return server.MakeServeContent(overviewURL, d)
		}
		if page := pages[uri]; page != nil {
			return func(w http.ResponseWriter, r *http.Request) {
				if newPage := maybeRedownloadPage(book, page.NotionID); newPage != nil {
					page = newPage
					pages[uri] = page
				}
				html := notionToHTML(page, book)
				page.BodyHTML = template.HTML(string(html))
				d := PageData{
					PageCommon:  getPageCommon(),
					Page:        page,
					Description: page.Title,
				}
				buildCreadcumb(book, page, &d)
				err := execTemplateToWriter("page.tmpl.html", d, w)
				if err != nil {
					logf(ctx(), "Failed to generate page %s in book %s\n", page.NotionID, book.Title)
				}
			}
		}
		for _, h := range handlers {
			if res := h.Get(uri); res != nil {
				return res
			}
		}
		return nil
	}

	policy := notionapi.PolicyCacheOnly

	// start generating urls in background
	booksWg.Add(1)
	go func() {
		booksSem <- true
		defer func() {
			<-booksSem
			booksWg.Done()
		}()
		logf(ctx(), "starting to build book '%s'\n", book.DirShort)
		timeStart := time.Now()
		defer func() {
			urls := getURLs()
			logf(ctx(), "finished building book '%s', %d urls, took %s\n", book.DirShort, len(urls), time.Since(timeStart))
		}()

		downloadBook(book, policy)

		buildBookPages(book)

		{
			dir := filepath.Join(book.NotionCacheDir, "img")
			urlPrefix := path.Join(baseURL, "img")
			addHandler(server.NewDirHandler(dir, urlPrefix, nil))
		}
		addHandler(genBookTOCSearchHandlerMust(book))
		{
			{
				fpath := filepath.Join("covers", book.CoverImageName)
				uri := book.CoverURL()
				filesHandler.AddFile(uri, fpath)
			}
			{
				fpath := filepath.Join("covers_small", book.CoverImageName)
				uri := book.CoverSmallURL()
				filesHandler.AddFile(uri, fpath)
			}
			{
				fpath := filepath.Join("covers", "twitter", book.CoverImageName)
				uri := book.CoverTwitterURL()
				filesHandler.AddFile(uri, fpath)
			}
		}

		ensureHTMLSuffix := func(s string) string {
			if !strings.HasSuffix(s, ".html") {
				return s + ".html"
			}
			return s
		}

		{
			// genPage
			addPage := func(page *Page) {
				uri := ensureHTMLSuffix(page.URL())
				pages[uri] = page
				for _, imagePath := range page.images {
					imageName := filepath.Base(imagePath)
					uri := page.ImageURL(imageName)
					dst := page.destImagePath(imageName)
					filesHandler.AddFile(uri, dst)
				}
			}

			mu.Lock()
			for _, chapter := range book.Chapters() {
				addPage(chapter)
				for _, article := range chapter.Pages {
					addPage(article)
				}
			}
			mu.Unlock()
		}
	}()
	return server.NewDynamicHandler(get, getURLs)
}

var (
	didBuildFrontEnd = false
)

func buildFrontend(forDev bool) {
	// only needs to do it once
	if didBuildFrontEnd {
		return
	}
	// this is to speed up local dev
	if !dirExists("node_modules") {
		cmd := exec.Command("npm", "install")
		runCmdMust(cmd)
	}
	if forDev {
		cmd := exec.Command("npm", "run", "build-dev")
		runCmdMust(cmd)
	} else {
		cmd := exec.Command("npm", "run", "build")
		runCmdMust(cmd)
	}
	didBuildFrontEnd = true
}

func buildFrontendDev() {
	buildFrontend(true)
}

func buildFrontendProd() {
	buildFrontend(false)
}

func buildServer(booksToProcess []*Book, forDev bool) *server.Server {
	logf(ctx(), "buildServer: %d books\n", len(booksToProcess))
	initBookHandlers()

	if forDev {
		buildFrontendDev()
	} else {
		buildFrontendProd()
	}
	filesHandler := server.NewFilesHandler()

	dir := filepath.Join(dirWwwGenerated, "svelte")
	filesHandler.AddFilesInDir(dir, "/s/", []string{"bundle.css", "bundle.js"})
	dir = filepath.Join("fe", "tmpl")
	filesHandler.AddFilesInDir(dir, "/s/", []string{"favicon.ico", "index.css", "main.css"})
	filesHandler.AddFilesInDir(dir, "/", []string{"ping.txt"})

	handlers := []server.Handler{filesHandler}
	h := genBooksIndexHandler(booksToProcess)
	handlers = append(handlers, h)

	server := &server.Server{
		Handlers:  handlers,
		Port:      httpPort,
		CleanURLS: true,
	}

	for _, book := range booksToProcess {
		h := genBookHandler(book)
		server.Handlers = append(server.Handlers, h)
	}

	return server
}
