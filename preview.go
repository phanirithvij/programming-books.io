package main

import (
	"html/template"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// return "" if didn't find a file
// uriPath is
func findFileForURL(uri *url.URL) string {
	dir := indexDestDir
	uriPath := uri.Path
	fileName := uriPath
	if uriPath == "/" {
		fileName = "index.html"
	}
	path := filepath.Join(dir, fileName)
	if fileExists(path) {
		return path
	}
	if fileExists(path + ".html") {
		return path + ".html"
	}
	if fileExists(path + "/index.html") {
		return path + "/"
	}
	logf(ctx(), "didn't find file or url '%s'\n", uriPath)
	return ""
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	//logf(ctx(), "uri: %s\n", r.URL.Path)
	path := findFileForURL(r.URL)
	if path != "" {
		http.ServeFile(w, r, path)
		return
	}
	path = filepath.Join(indexDestDir, "404.html")
	d := readFileMust(path)
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write(d)
}

// https://blog.gopheracademy.com/advent-2016/exposing-go-on-the-internet/
func makeHTTPServerOnDemand() *http.Server {
	mux := &http.ServeMux{}

	mux.HandleFunc("/", handleIndex)

	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second, // introduced in Go 1.8
		Handler:      mux,
	}
	return srv
}

func previewWebsite() {
	httpSrv := makeHTTPServerOnDemand()
	httpSrv.Addr = "127.0.0.1:8183"

	go func() {
		err := httpSrv.ListenAndServe()
		// mute error caused by Shutdown()
		if err == http.ErrServerClosed {
			err = nil
		}
		must(err)
		logf(ctx(), "HTTP server shutdown gracefully\n")
	}()
	logf(ctx(), "Started listening on %s\n", httpSrv.Addr)
	openBrowser("http://" + httpSrv.Addr)

	waitForCtrlC()
}

func staticFileHandlers(dir string, files []string) []Handler {
	var res []Handler
	for _, f := range files {
		uri := "/s/" + f
		path := filepath.Join(dir, f)
		panicIf(!fileExists(path))
		h := NewFileHandler(path, uri)
		res = append(res, h)
	}
	return res
}

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

func previewWebsite2(booksToProcess []*Book) {
	logf(ctx(), "previewWebsite2\n")
	flgReloadTemplates = true
	flgNoDownload = false
	for _, book := range booksToProcess {
		initBook(book)
	}
	buildFrontend()
	h := staticFileHandlers(filepath.Join("www", "gen"), []string{"bundle.css", "bundle.js"})
	handlers := h
	h = staticFileHandlers(filepath.Join("fe", "tmpl"), []string{"favicon.ico", "index.css", "main.css"})
	handlers = append(handlers, h...)
	h = genBooksIndex2(allBooks)
	handlers = append(handlers, h...)
	initBookHandlers()
	for _, book := range booksToProcess {
		h := genBookHandler(book)
		handlers = append(handlers, h)
	}

	server := &ServerConfig{
		Handlers:  handlers,
		Port:      9003,
		CleanURLS: true,
	}
	waitSignal := StartServer(server)
	waitSignal()
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
	var urls []string
	var handlers []Handler
	var mu sync.Mutex
	pages := map[string]*Page{}
	getURLs := func() []string {
		mu.Lock()
		defer mu.Unlock()

		return urls
	}
	baseURL := book.URL()
	indexURL := path.Join(baseURL, "index.html")
	book404URL := path.Join(baseURL, "404.html")
	overviewURL := path.Join(baseURL, "overview.html")

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
			logf(ctx(), "finished building book '%s', %d urls, took %s\n", book.DirShort, len(urls), time.Since(timeStart))
		}()

		downloadBook(book)

		buildBookPages(book)

		addSitemapURL(book, book.CanonnicalURL())
		addHandler := func(h Handler) {
			mu.Lock()
			handlers = append(handlers, h)
			urls = append(urls, h.URLS()...)
			mu.Unlock()
		}
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
				h := NewFileHandler(fpath, uri)
				addHandler(h)
			}
			{
				fpath := filepath.Join("covers", "twitter", book.CoverImageName)
				uri := path.Join(baseURL, "covers", "twitter", book.CoverImageName)
				h := NewFileHandler(fpath, uri)
				addHandler(h)
			}
		}

		{
			// genPage
			addPageImages := func(page *Page) {
				for _, imagePath := range page.images {
					imageName := filepath.Base(imagePath)
					uri := page.ImageURL(imageName)
					dst := page.destImagePath(imageName)
					h := NewFileHandler(dst, uri)
					addHandler(h)
				}
			}

			mu.Lock()
			for _, chapter := range book.Chapters() {
				pages[chapter.URL()] = chapter
				addSitemapURL(book, chapter.CanonnicalURL())
				addPageImages(chapter)
				for _, article := range chapter.Pages {
					pages[article.URL()] = article
					addSitemapURL(book, article.CanonnicalURL())
					addPageImages(article)
				}
			}
			mu.Unlock()
		}

		mu.Lock()
		defer mu.Unlock()
		urls = append(urls, indexURL, book404URL)
	}()
	return NewDynamicHandler(get, getURLs)
}

/*

	for _, chapter := range book.Chapters() {
		_ = genPage(book, chapter, nil)
	}

	writeSitemap(book)
*/
