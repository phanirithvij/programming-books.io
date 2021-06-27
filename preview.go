package main

import (
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/kjk/u"
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
	if u.FileExists(path) {
		return path
	}
	if u.FileExists(path + ".html") {
		return path + ".html"
	}
	if u.FileExists(path + "/index.html") {
		return path + "/"
	}
	logf("didn't find file or url '%s'\n", uriPath)
	return ""
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	logf("uri: %s\n", r.URL.Path)
	path := findFileForURL(r.URL)
	if path != "" {
		http.ServeFile(w, r, path)
		return
	}
	path = filepath.Join(indexDestDir, "404.html")
	d := u.ReadFileMust(path)
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
		logf("HTTP server shutdown gracefully\n")
	}()
	logf("Started listening on %s\n", httpSrv.Addr)
	u.OpenBrowser("http://" + httpSrv.Addr)

	u.WaitForCtrlC()
}
