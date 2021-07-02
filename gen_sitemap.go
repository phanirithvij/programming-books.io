package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

func isFullURL(uri string) bool {
	return strings.HasPrefix(uri, "https://") || strings.HasPrefix(uri, "http://")
}

func addSitemapURL(b *Book, uri string) {
	if !isFullURL(uri) {
		uri = urlJoin(siteBaseURL, uri)
	}
	b.muSitemapURLS.Lock()
	b.sitemapURLS[uri] = struct{}{}
	b.muSitemapURLS.Unlock()
}

const (
	sitemapTmpl = `User-agent: *
Disallow:

Sitemap: %s
`
)

func writeSitemap(b *Book) {
	// http://www.advancedhtml.co.uk/robots-sitemaps.htm
	sitemapURL := urlJoin(siteBaseURL, "sitemap.txt")
	robotsTxt := fmt.Sprintf(sitemapTmpl, sitemapURL)
	robotsTxtPath := filepath.Join(b.DirOnDisk, "robots.txt")
	err := ioutil.WriteFile(robotsTxtPath, []byte(robotsTxt), 0644)
	must(err)

	addSitemapURL(b, "/")
	//addSitemapURL(b, "about")

	var urls []string
	for uri := range b.sitemapURLS {
		urls = append(urls, uri)
	}
	sort.Strings(urls)
	s := strings.Join(urls, "\n")
	sitemapPath := filepath.Join(b.DirOnDisk, "sitemap.txt")
	err = ioutil.WriteFile(sitemapPath, []byte(s), 0644)
	must(err)
}
