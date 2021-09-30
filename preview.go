package main

import (
	"net/http"
	"net/url"
	"path/filepath"
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

func serverURLS() []string {
	files := []string{
		"/index.html",
		"/index-grid.html",
	}
	return files
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
	}
	return nil
}

/*
	gen404Indexl(indexDestDir)
	_ = genAbout(indexDestDir, nil)
	_ = genFeedback(indexDestDir, nil)
*/

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

	server := &ServerConfig{
		Handlers:  handlers,
		Port:      9003,
		CleanURLS: true,
	}
	waitSignal := StartServer(server)
	waitSignal()
}
