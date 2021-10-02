package main

import (
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"path/filepath"
	"sync"
)

const (
	siteBaseURL   = "https://www.programming-books.io"
	gitHubBaseURL = "https://github.com/essentialbooks/tools"
	notionBaseURL = "https://notion.so/"
)

var (
	templates *template.Template

	funcMap = template.FuncMap{
		"inc": funcInc,
	}
)

func funcInc(i int) int {
	return i + 1
}

var (
	flgReloadTemplates = false
	muTemplates        sync.Mutex
)

func loadTemplatesMust() *template.Template {
	muTemplates.Lock()
	defer muTemplates.Unlock()

	// we reload templates in preview mode
	if !flgReloadTemplates && templates != nil {
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

	must(createDirForFile(path))
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
		Analytics: "",
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

func genBookIndexHTML(book *Book, w io.Writer) error {
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
