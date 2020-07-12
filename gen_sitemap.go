package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	muSitemapURLS sync.Mutex
	sitemapURLS   = map[string]struct{}{}
)

func clearSitemapURLS() {
	sitemapURLS = make(map[string]struct{})
}

func isFullURL(uri string) bool {
	return strings.HasPrefix(uri, "https://") || strings.HasPrefix(uri, "http://")
}

func addSitemapURL(b *Book, uri string) {
	if !isFullURL(uri) {
		uri = urlJoin(b.BaseURL(), uri)
	}
	muSitemapURLS.Lock()
	sitemapURLS[uri] = struct{}{}
	muSitemapURLS.Unlock()
}

const (
	sitemapTmpl = `User-agent: *
Disallow:

Sitemap: %s
`
)

// http://www.advancedhtml.co.uk/robots-sitemaps.htm
func writeRobots(b *Book) {
	sitemapURL := urlJoin(b.BaseURL(), "sitemap.txt")
	robotsTxt := fmt.Sprintf(sitemapTmpl, sitemapURL)
	robotsTxtPath := filepath.Join("www", "robots.txt")
	err := ioutil.WriteFile(robotsTxtPath, []byte(robotsTxt), 0644)
	must(err)
}

func writeSitemap(b *Book) {
	writeRobots(b)

	addSitemapURL(b, "/")
	addSitemapURL(b, "about")

	var urls []string
	for uri := range sitemapURLS {
		urls = append(urls, uri)
	}
	sort.Strings(urls)
	s := strings.Join(urls, "\n")
	sitemapPath := filepath.Join("www", "sitemap.txt")
	err := ioutil.WriteFile(sitemapPath, []byte(s), 0644)
	must(err)

	clearSitemapURLS()
}
