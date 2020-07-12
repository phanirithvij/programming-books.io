package main

import (
	"html/template"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/kjk/u"
)

const (
	// top-level directory where .html files are generated
	destDir = "www"

	gitHubBaseURL = "https://github.com/essentialbooks/books"
	notionBaseURL = "https://notion.so/"
	siteBaseURL   = "https://www.programming-books.io"
)

var (
	tmplDir = filepath.Join("fe", "tmpl")

	// directory where generated .html files for books are
	destEssentialDir = filepath.Join("www")

	templates *template.Template

	funcMap = template.FuncMap{
		"inc":      funcInc,
		"optimize": funcOptimizeAsset,
	}
)

var (
	// sha1 of original content to url of optimized
	// content
	hashToOptimizedURL = map[string]string{}
)

func funcOptimizeAsset(url string) string {
	// url is like /s/app.js which we convert to a file
	// tmpl/app.js
	name := strings.TrimPrefix(url, "/s/")
	srcPath := filepath.Join("fe", "tmpl", name)
	d, err := ioutil.ReadFile(srcPath)
	if err != nil {
		// for bundle.js and bundle.css
		srcPath = filepath.Join("www", "gen", name)
		d, err = ioutil.ReadFile(srcPath)
	}
	u.Must(err)
	srcSha1Hex := u.Sha1HexOfBytes(d)
	if newURL := hashToOptimizedURL[srcSha1Hex]; newURL != "" {
		return newURL
	}

	dstSha1Hex := u.Sha1HexOfBytes(d)
	dstName := nameToSha1Name(name, dstSha1Hex)
	dstPath := filepath.Join("www", "s", dstName)
	dstURL := "/s/" + dstName
	err = ioutil.WriteFile(dstPath, d, 0644)
	u.Must(err)
	logf("Copied %s => %s\n", srcPath, dstPath)
	hashToOptimizedURL[srcSha1Hex] = dstURL
	return dstURL
}

func funcInc(i int) int {
	return i + 1
}
