package main

import (
	"fmt"

	"github.com/kjk/u"
)

var srcFiles = u.MakeAllowedFileFilterForExts(".go", ".js", ".html", ".css", ".svelte")
var excludeDirs = u.MakeExcludeDirsFilter("books", ".vscode", ".github", ".idea", "cache", "code", "covers", "log", "mdfmt", "node_modules", "pkg", "essential", "s")
var allFiles = u.MakeFilterAnd(srcFiles, excludeDirs)

func doLineCount() int {
	stats := u.NewLineStats()
	err := stats.CalcInDir(".", allFiles, true)
	if err != nil {
		fmt.Printf("doLineCount: stats.wcInDir() failed with '%s'\n", err)
		return 1
	}
	u.PrintLineStats(stats)
	return 0
}
