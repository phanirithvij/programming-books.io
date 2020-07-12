package main

import (
	"fmt"
	"html/template"
)

var (
	googleAnalytics template.HTML
	doMinify        bool

	notionAuthToken string

	// when downloading pages from the server, count total number of
	// downloaded and those from cache
	nTotalDownloaded int
	nTotalFromCache  int
)

func isPreview() bool {
	return flgPreviewStatic || flgPreviewOnDemand
}

var (
	// url or id of the page to rebuild
	flgNoUpdateOutput bool
	// if true, disables notion cache, forcing re-download of notion page
	// even if cached verison on disk exits
	flgDisableNotionCache       bool
	flgPreviewStatic            bool
	flgPreviewOnDemand          bool
	flgReportStackOverflowLinks bool
	// if true, disables downloading pages
	flgNoDownload     bool
	flgGistRedownload bool
)

func main() {
	fmt.Printf("Hello\n")
}
