package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/kjk/notionapi"
	"github.com/kjk/u"
)

// Book represents a book
type Book struct {
	Title     string // "Go", "jQuery" etcc
	TitleLong string // "Essential Go", "Essential jQuery" etc.

	// used by page index. defaults to: "<b>${TitleLong}</b> is a free book about ${Title} programming language."
	summary string

	NotionStartPageID string
	RootPage          *Page   // a tree of pages
	cachedPages       []*Page // pages flattened into an array

	idToPage map[string]*Page

	DirShort       string // directory name for the book e.g. "go"
	DirOnDisk      string // full directory on disk, ${generated}/www/essential/${DirShort}
	DirCache       string // full path of sub-directory "cache"
	NotionCacheDir string

	// generated toc javascript data
	tocData []byte
	// url of combined tocData and app.js
	AppJSURL string

	// name of a file in covers/ directory
	// e.g. "Python.png"
	CoverImageName string

	client *notionapi.Client
	// cache related
	cache *Cache

	muSitemapURLS sync.Mutex
	sitemapURLS   map[string]struct{}
}

func (b *Book) cachePath() string {
	return filepath.Join(b.DirCache, "cache.txt")
}

// this is where html etc. files for a book end up
func (b *Book) destDir() string {
	return b.DirOnDisk
}

// URL returns url of the book, used in index.tmpl.html
func (b *Book) URL() string {
	return "/"
}

// Summary returns book summary
func (b *Book) Summary() template.HTML {
	if b.summary == "" {
		b.summary = fmt.Sprintf("<b>%s</b> is a free book about %s programming language.", b.TitleLong, b.Title)
	}
	return template.HTML(b.summary)
}

func (b *Book) BaseURL() string {
	//return "https://www.programming-books.io/essential/" + b.DirShort
	return "/essential/" + b.DirShort
}

// CanonnicalURL returns full url including host
func (b *Book) CanonnicalURL() string {
	return urlJoin(b.BaseURL(), b.URL())
}

// ShareOnTwitterText returns text for sharing on twitter
func (b *Book) ShareOnTwitterText() string {
	return fmt.Sprintf(`"%s" - a free programming book`, b.TitleLong)
}

// CoverURL returns url to cover image
func (b *Book) CoverURL() string {
	u.PanicIf(b.CoverImageName == "")
	return fmt.Sprintf("/covers/%s", b.CoverImageName)
}

// CoverSmallURL returns url to small cover image
func (b *Book) CoverSmallURL() string {
	u.PanicIf(b.CoverImageName == "")
	return fmt.Sprintf("/covers_small/%s", b.CoverImageName)
}

// CoverFullURL returns a URL for the cover including host
func (b *Book) CoverFullURL() string {
	return urlJoin(b.BaseURL(), b.CoverURL())
}

// CoverTwitterFullURL returns a URL for the cover including host
func (b *Book) CoverTwitterFullURL() string {
	u.PanicIf(b.CoverImageName == "")
	coverURL := fmt.Sprintf("/covers/twitter/%s", b.CoverImageName)
	return urlJoin(b.BaseURL(), coverURL)
}

// Chapters returns pages that are top-level chapters
func (b *Book) Chapters() []*Page {
	return b.RootPage.Pages
}

// GetAllPages returns all pages, flattened
func (b *Book) GetAllPages() []*Page {
	// to prevent infinite recursion if pages show up twice (shouldn't happen)
	if len(b.cachedPages) > 0 {
		return b.cachedPages
	}
	if b.RootPage == nil {
		return nil
	}
	seen := map[*Page]bool{}
	pages := []*Page{b.RootPage}
	curr := 0
	for curr < len(pages) {
		page := pages[curr]
		curr++
		if seen[page] {
			continue
		}
		seen[page] = true
		for _, p := range page.Pages {
			p.Parent = page
		}
		pages = append(pages, page.Pages...)
	}
	return pages
}

// PagesCount returns total number of articles
func (b *Book) PagesCount() int {
	return len(b.GetAllPages()) - 1 // don't count top page
}

// ChaptersCount returns number of chapters
func (b *Book) ChaptersCount() int {
	return len(b.RootPage.Pages)
}

func updateBookAppJS(book *Book) {
	srcName := fmt.Sprintf("app-%s.js", book.DirShort)
	d := book.tocData

	sha1Hex := u.Sha1HexOfBytes(d)
	name := nameToSha1Name(srcName, sha1Hex)
	dst := filepath.Join(book.DirOnDisk, "s", name)
	err := ioutil.WriteFile(dst, d, 0644)
	maybePanicIfErr(err)
	if err != nil {
		return
	}
	book.AppJSURL = "/s/" + name
	logf("Created %s\n", dst)
}

func calcPageHeadings(page *Page) {
	var headings []*HeadingInfo
	cb := func(block *notionapi.Block) {
		switch block.Type {
		case notionapi.BlockHeader, notionapi.BlockSubHeader, notionapi.BlockSubSubHeader:
			// do nothing, those are headers
		default:
			// not a header, so exit
			return
		}
		id := notionapi.ToNoDashID(block.ID)
		s := getInlinesPlain(block.InlineContent)
		h := &HeadingInfo{
			Text: s,
			ID:   id,
		}
		headings = append(headings, h)
	}
	blocks := []*notionapi.Block{page.NotionPage.Root()}
	notionapi.ForEachBlock(blocks, cb)
	page.Headings = headings
}

func (b *Book) afterPageDownload(page *notionapi.Page) error {
	id := toNoDashID(page.ID)
	p := &Page{
		NotionPage: page,
		NotionID:   id,
		Book:       b,
	}
	b.idToPage[id] = p
	evalCodeSnippetsForPage(p)
	downloadImages(b, p)
	calcPageHeadings(p)
	return nil
}

func initBook(book *Book) {
	book.DirOnDisk = filepath.Join(gDestDir, "www", "essential", book.DirShort)
	book.DirCache = filepath.Join("books", book.DirShort, "cache")
	book.NotionCacheDir = filepath.Join(book.DirCache, "notion")
	currBookDir = book.DirOnDisk
	// cache is only valid for the book
	hashToOptimizedURL = map[string]string{}
	dir := book.NotionCacheDir
	u.CreateDirMust(dir)
	if flgClean {
		dir = filepath.Join(book.DirOnDisk)
		os.RemoveAll(dir)
	}
	logf("Created '%s' for book '%s'\n", dir, book.Title)
	book.idToPage = map[string]*Page{}
	book.sitemapURLS = map[string]struct{}{}
	book.cache = loadCache(book)
}

func downloadBook(book *Book) {
	logf("Loading %s...\n", book.Title)
	nProcessed = 0
	nNotionPagesFromCache = 0
	nDownloadedPages = 0

	book.client = newNotionClient()
	cacheDir := book.NotionCacheDir
	d, err := notionapi.NewCachingClient(cacheDir, book.client)
	must(err)
	d.EventObserver = eventObserver
	d.RedownloadNewerVersions = !flgNoDownload
	d.NoReadCache = flgDisableNotionCache

	startPageID := book.NotionStartPageID
	pages, err := d.DownloadPagesRecursively(startPageID, book.afterPageDownload)
	must(err)
	nPages := len(pages)
	logf("Got %d pages for %s, downloaded: %d, from cache: %d\n", nPages, book.Title, nDownloadedPages, nNotionPagesFromCache)
}
