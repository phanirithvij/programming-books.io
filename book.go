package main

import "github.com/kjk/notionapi"

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

	Dir string // directory name for the book e.g. "go"

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
}
