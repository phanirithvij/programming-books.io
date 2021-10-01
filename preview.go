package main

import (
	"html/template"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

func serveStart(w http.ResponseWriter, r *http.Request, uri string) {
	if r == nil {
		return
	}
	ct := mimeTypeFromFileName(uri)
	w.Header().Add("Content-Type", ct)
	w.WriteHeader(http.StatusOK) // 200
}

func serverGet(uri string) func(w http.ResponseWriter, r *http.Request) {
	//logf(ctx(), "serverGet: '%s'\n", uri)

	books := allBooks
	/*
		writeData := func(w http.ResponseWriter, d []byte, err error) {
			must(err)
			_, err = w.Write(d)
			must(err)
		}
	*/
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
			gen404Indexl("", w)
		}
	case "/about.html":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
			serveStart(w, r, uri)
			genAbout("", w)
		}
	case "/feedback.html":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
			serveStart(w, r, uri)
			genFeedback("", w)
		}
	}

	return nil
}
func serverURLS() []string {
	files := []string{
		"/index.html",
		"/index-grid.html",
		"/404.html",
		"/about.html",
		"/feedback.html",
	}
	return files
}

func genBooksIndex2(books []*Book) []Handler {
	var res []Handler

	h := NewDirHandler("covers", "/covers/", nil)
	res = append(res, h)
	h = NewDirHandler("covers_small", "/covers_small/", nil)
	res = append(res, h)

	h2 := NewDynamicHandler(serverGet, serverURLS)
	res = append(res, h2)
	return res
}

func buildServer(booksToProcess []*Book, forDev bool) *ServerConfig {
	logf(ctx(), "buildServer: %d books\n", len(booksToProcess))
	for _, book := range booksToProcess {
		initBook(book)
	}
	initBookHandlers()

	if forDev {
		buildFrontendDev()
	} else {
		buildFrontendProd()
	}
	filesHandler := NewFilesHandler()

	filesHandler.AddFilesInDir(filepath.Join("www", "gen"), []string{"bundle.css", "bundle.js"})
	filesHandler.AddFilesInDir(filepath.Join("fe", "tmpl"), []string{"favicon.ico", "index.css", "main.css"})
	handlers := []Handler{filesHandler}
	h := genBooksIndex2(allBooks)
	handlers = append(handlers, h...)
	for _, book := range booksToProcess {
		h := genBookHandler(book)
		handlers = append(handlers, h)
	}

	server := &ServerConfig{
		Handlers:  handlers,
		Port:      9003,
		CleanURLS: true,
	}
	return server
}

func previewWebsite(booksToProcess []*Book) {
	logf(ctx(), "previewWebsite\n")
	timeStart := time.Now()
	flgReloadTemplates = true
	flgNoDownload = true
	server := buildServer(booksToProcess, true)
	go func() {
		booksWg.Wait()
		// TODO: mutex protection
		nPages := 0
		for _, h := range server.Handlers {
			nPages += len(h.URLS())
		}
		logf(ctx(), "previewWebsite: finished %d urls in %s\n", nPages, time.Since(timeStart))
	}()

	waitSignal := StartServer(server)
	waitSignal()
}

func uploadZipToInstantPreviewMust(zipData []byte) string {
	timeStart := time.Now()
	uri := "https://www.instantpreview.dev/upload"
	res, err := httpPost(uri, zipData)
	must(err)
	uri = string(res)
	sizeStr := formatSize(int64(len(zipData)))
	logf(ctx(), "uploaded under: %s, zip file size: %s in: %s\n", uri, sizeStr, time.Since(timeStart))
	return uri
}

func previewToInsantPreview(booksToProcess []*Book) {
	logf(ctx(), "previewToInsantPreview: %d books\n", len(booksToProcess))
	timeStart := time.Now()
	flgReloadTemplates = false
	flgNoDownload = true
	server := buildServer(booksToProcess, false)
	booksWg.Wait()
	nPages := 0
	for _, h := range server.Handlers {
		nPages += len(h.URLS())
	}
	logf(ctx(), "previewToInsantPreview: finished %d urls in %s\n", nPages, time.Since(timeStart))
	zipData, err := WriteServerFilesToZip(server.Handlers)
	must(err)
	timeStart = time.Now()
	uri := uploadZipToInstantPreviewMust(zipData)
	logf(ctx(), "previewToInsantPreview: uploaded zip of size %s in %s\n%s\n", formatSize(int64(len(zipData))), time.Since(timeStart), uri)
}

func genToDir(booksToProcess []*Book, dir string) {
	logf(ctx(), "genToDir: '%s'\n", dir)
	timeStart := time.Now()
	flgReloadTemplates = false
	flgNoDownload = true
	server := buildServer(booksToProcess, false)
	booksWg.Wait()
	nPages := 0
	for _, h := range server.Handlers {
		nPages += len(h.URLS())
	}
	logf(ctx(), "genToDir: finished %d urls in %s\n", nPages, time.Since(timeStart))
	//must(os.RemoveAll(dir))
	nFiles, totalSize := WriteServerFilesToDir(dir, server.Handlers)
	logf(ctx(), "genToDir: wrote %d files of size %s\n", nFiles, formatSize(totalSize))
}

var booksSem chan bool
var booksWg sync.WaitGroup

// call before genBookHandler
func initBookHandlers() {
	nThreads := runtime.NumCPU()
	logf(ctx(), "initBookHandler: %d threads\n", nThreads)
	booksSem = make(chan bool, nThreads)
}

// returns immediately but builds the handler in the background
func genBookHandler(book *Book) Handler {
	var handlers []Handler
	var filesHandler *FilesHandler
	var mu sync.Mutex
	pages := map[string]*Page{} // maps url to Page

	addHandler := func(h Handler) {
		mu.Lock()
		handlers = append(handlers, h)
		mu.Unlock()
	}
	filesHandler = NewFilesHandler()
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
			return makeServeContent(d, overviewURL)
		}
		if page := pages[uri]; page != nil {
			return func(w http.ResponseWriter, r *http.Request) {
				html := notionToHTML(page, book)
				page.BodyHTML = template.HTML(string(html))
				d := PageData{
					PageCommon:  getPageCommon(),
					Page:        page,
					Description: page.Title,
				}
				buildCreadcumb(book, page, &d)
				path := page.destFilePath()
				err := execTemplate("page.tmpl.html", d, path, w)
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

		downloadBook(book)

		buildBookPages(book)

		addSitemapURL(book, book.CanonnicalURL())

		{
			// copyImages
			dir := filepath.Join(book.NotionCacheDir, "img")
			urlPrefix := path.Join(baseURL, "img")
			addHandler(NewDirHandler(dir, urlPrefix, nil))
		}
		// genBookTOCSearchMust
		addHandler(genBookTOCSearchHandlerMust(book))
		{
			// copyCover
			{
				fpath := filepath.Join("covers", book.CoverImageName)
				uri := path.Join(baseURL, "covers", book.CoverImageName)
				filesHandler.AddFile(uri, fpath)
			}
			{
				fpath := filepath.Join("covers", "twitter", book.CoverImageName)
				uri := path.Join(baseURL, "covers", "twitter", book.CoverImageName)
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
				addSitemapURL(book, page.CanonnicalURL())
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
	return NewDynamicHandler(get, getURLs)
}

/*
	writeSitemap(book)
*/
