package main

import (
	"html/template"
	"io"
	"path/filepath"
	"sync"
)

const (
	siteBaseURL   = "https://www.programming-books.io"
	gitHubBaseURL = "https://github.com/essentialbooks/tools"
	notionBaseURL = "https://notion.so/"
)

var (
	flgReloadTemplates = false
	muTemplates        sync.Mutex
	templates          *template.Template

	// only a dummpy function to show how to add more
	funcMap = template.FuncMap{
		"inc": funcInc,
	}
)

func funcInc(i int) int {
	return i + 1
}

func execTemplateToWriter(name string, data interface{}, w io.Writer) error {
	muTemplates.Lock()
	defer muTemplates.Unlock()

	// we reload templates in preview mode
	if flgReloadTemplates || templates == nil {
		pattern := filepath.Join("fe", "tmpl", "*.tmpl.html")
		var err error
		templates, err = template.New("").Funcs(funcMap).ParseGlob(pattern)
		//templates, err = template.ParseGlob(pattern)
		must(err)
	}
	tmpl := templates.Lookup(name)
	panicIf(tmpl == nil, "no template named '%s'", name)
	return tmpl.Execute(w, data)
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
	return execTemplateToWriter("book_index.tmpl.html", data, w)
}

func genBook404(book *Book, w io.Writer) error {
	data := struct {
		PageCommon
		Book *Book
	}{
		PageCommon: getPageCommon(),
		Book:       book,
	}
	return execTemplateToWriter("404.tmpl.html", data, w)
}
