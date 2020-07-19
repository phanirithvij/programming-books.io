package main

import (
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/u"
)

const (
	// top-level directory where .html files are generated
	destDir = "www"

	gitHubBaseURL = "https://github.com/essentialbooks/books"
	notionBaseURL = "https://notion.so/"
)

var (
	tmplDir = filepath.Join("fe", "tmpl")

	// directory where generated .html files for books are
	destEssentialDir = "www"
	currBookDir      string // a bit of a hack

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
	must(err)
	srcSha1Hex := u.Sha1HexOfBytes(d)
	if newURL := hashToOptimizedURL[srcSha1Hex]; newURL != "" {
		return newURL
	}

	dstSha1Hex := u.Sha1HexOfBytes(d)
	dstName := nameToSha1Name(name, dstSha1Hex)
	dstPath := filepath.Join(currBookDir, "www", "s", dstName)
	dstURL := "/s/" + dstName
	err = ioutil.WriteFile(dstPath, d, 0644)
	must(err)
	logf("Copied %s => %s\n", srcPath, dstPath)
	hashToOptimizedURL[srcSha1Hex] = dstURL
	return dstURL
}

func funcInc(i int) int {
	return i + 1
}

func loadTemplatesMust() *template.Template {
	// we reload templates in preview mode
	if templates != nil {
		return templates
	}
	pattern := filepath.Join("fe", "tmpl", "*.tmpl.html")
	var err error
	templates, err = template.New("").Funcs(funcMap).ParseGlob(pattern)
	//templates, err = template.ParseGlob(pattern)
	must(err)
	templates.Funcs(funcMap)
	return templates
}

func execTemplateToFileSilentMaybeMust(name string, data interface{}, path string) error {
	var errToReturn error
	tmpl := loadTemplatesMust()
	if tmpl == nil {
		return nil
	}
	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, name, data)
	maybePanicIfErr(err)

	d := buf.Bytes()
	err = ioutil.WriteFile(path, d, 0644)
	maybePanicIfErr(err)
	return errToReturn
}

func execTemplateToFileMaybeMust(name string, data interface{}, path string) error {
	return execTemplateToFileSilentMaybeMust(name, data, path)
}

func execTemplateToWriter(name string, data interface{}, w io.Writer) error {
	tmpl := loadTemplatesMust()
	return tmpl.ExecuteTemplate(w, name, data)
}

// PageCommon is a common information for most pages
type PageCommon struct {
	Analytics template.HTML
}

func getPageCommon() PageCommon {
	return PageCommon{
		Analytics: googleAnalytics,
	}
}

func splitBooks(books []*Book) ([]*Book, []*Book) {
	var left []*Book
	var right []*Book
	for i, book := range books {
		if i%2 == 0 {
			left = append(left, book)
		} else {
			right = append(right, book)
		}
	}
	return left, right
}

func execTemplate(tmplName string, d interface{}, path string, w io.Writer) error {
	// this code path is for the preview on demand server
	if w != nil {
		return execTemplateToWriter(tmplName, d, w)
	}

	// this code path is for generating static files
	_ = execTemplateToFileMaybeMust(tmplName, d, path)
	return nil
}

type Breadcrumb struct {
	URL   string
	Title string
}

type PageData struct {
	PageCommon
	*Page
	Description string
	Breadcrumbs []Breadcrumb
}

func buildCreadcumb(book *Book, page *Page, d *PageData) {
	page = page.Parent

	var a []Breadcrumb
	for page != nil && page.Parent != nil {
		b := Breadcrumb{
			Title: page.Title,
			URL:   page.URL(),
		}
		a = append(a, b)
		page = page.Parent
	}

	b := Breadcrumb{
		Title: book.Title,
		URL:   book.URL(),
	}
	a = append(a, b)

	// they were added in reverse order
	n := len(a)
	for i := 0; i < n/2; i++ {
		end := n - 1 - i
		a[i], a[end] = a[end], a[i]
	}
	d.Breadcrumbs = a
}

func genPage(book *Book, page *Page, w io.Writer) error {
	if w == nil {
		addSitemapURL(book, page.CanonnicalURL())
		for _, article := range page.Pages {
			_ = genPage(book, article, nil)
		}
	}

	d := PageData{
		PageCommon:  getPageCommon(),
		Page:        page,
		Description: page.Title,
	}
	buildCreadcumb(book, page, &d)

	path := page.destFilePath()
	err := execTemplate("page.tmpl.html", d, path, w)
	if err != nil {
		logf("Failed to minify page %s in book %s\n", page.NotionID, book.Title)
	}
	for _, imagePath := range page.images {
		imageName := filepath.Base(imagePath)
		dst := page.destImagePath(imageName)
		_ = copyFileMaybeMust(dst, imagePath)
	}
	return err
}

func bookPagesToHTML(book *Book) {
	nProcessed := 0
	pages := book.GetAllPages()
	for _, page := range pages {
		html := notionToHTML(page, book)
		page.BodyHTML = template.HTML(string(html))
		nProcessed++
	}
	logf("bookPagesToHTML: processed %d pages for book %s\n", nProcessed, book.TitleLong)
}

func genBookIndex(book *Book, w io.Writer) error {
	data := struct {
		PageCommon
		Book *Book
	}{
		PageCommon: getPageCommon(),
		Book:       book,
	}

	path := filepath.Join(book.destDir(), "index.html")
	return execTemplate("book_index.tmpl.html", data, path, w)
}

func genBook404(book *Book, w io.Writer) error {
	data := struct {
		PageCommon
		Book *Book
	}{
		PageCommon: getPageCommon(),
		Book:       book,
	}

	path := filepath.Join(book.destDir(), "404.html")
	return execTemplate("404.tmpl.html", data, path, nil)
}

func copyVercelRoutes(book *Book) {
	src := "vercel.json"
	dst := filepath.Join(book.DirOnDisk, "www", "vercel.json")
	u.CopyFileMust(dst, src)
	logf("Copied '%s' => '%s'\n", src, dst)
}

func copyCover(book *Book) {
	{
		src := filepath.Join("covers", book.CoverImageName)
		dst := filepath.Join(book.DirOnDisk, "www", "covers", book.CoverImageName)
		u.CreateDirForFileMust(dst)
		u.CopyFileMust(dst, src)
		logf("Copied '%s' => '%s'\n", src, dst)
	}

	{
		src := filepath.Join("covers", "twitter", book.CoverImageName)
		dst := filepath.Join(book.DirOnDisk, "www", "covers", "twitter", book.CoverImageName)
		u.CreateDirForFileMust(dst)
		u.CopyFileMust(dst, src)
		logf("Copied '%s' => '%s'\n", src, dst)
	}
}

func genBook(book *Book) {
	logf("Started genering book %s\n", book.Title)
	timeStart := time.Now()

	copyImages(book)
	bookPagesToHTML(book)

	genBookTOCSearchMust(book)

	// generate index.html for the book
	err := os.MkdirAll(book.destDir(), 0755)
	maybePanicIfErr(err)
	if err != nil {
		return
	}

	_ = genBookIndex(book, nil)

	// TODO: per-book 404 should link to top of book, not top of website
	_ = genBook404(book, nil)

	addSitemapURL(book, book.CanonnicalURL())

	for _, chapter := range book.Chapters() {
		_ = genPage(book, chapter, nil)
	}

	copyCover(book)
	copyVercelRoutes(book)

	logf("Generated book '%s' in %s\n", book.Title, time.Since(timeStart))
}
