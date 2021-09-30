package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/kjk/u"
)

func ctx() context.Context {
	return context.Background()
}

func panicIf(cond bool, args ...interface{}) {
	if !cond {
		return
	}
	s := "condition failed"
	if len(args) > 0 {
		s = fmt.Sprintf("%s", args[0])
		if len(args) > 1 {
			s = fmt.Sprintf(s, args[1:]...)
		}
	}
	panic(s)
}

func panicIfErr(err error, args ...interface{}) {
	if err == nil {
		return
	}
	s := err.Error()
	if len(args) > 0 {
		s = fmt.Sprintf("%s", args[0])
		if len(args) > 1 {
			s = fmt.Sprintf(s, args[1:]...)
		}
	}
	panic(s)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// whitelisted characters valid in url
func validateRune(c rune) byte {
	if c >= 'a' && c <= 'z' {
		return byte(c)
	}
	if c >= '0' && c <= '9' {
		return byte(c)
	}
	if c == '-' || c == '_' || c == '.' || c == ' ' {
		return '-'
	}
	return 0
}

func charCanRepeat(c byte) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	return false
}

// urlify generates safe url from tile by removing hazardous characters
func urlify(title string) string {
	s := strings.TrimSpace(title)
	s = strings.ToLower(s)
	var res []byte
	for _, r := range s {
		c := validateRune(r)
		if c == 0 {
			continue
		}
		// eliminute duplicate consequitive characters
		var prev byte
		if len(res) > 0 {
			prev = res[len(res)-1]
		}
		if c == prev && !charCanRepeat(c) {
			continue
		}
		res = append(res, c)
	}
	s = string(res)
	if len(s) > 128 {
		s = s[:128]
	}
	s = strings.TrimLeft(s, "-")
	s = strings.TrimRight(s, "-")
	return s
}

func openForAppend(path string) *os.File {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	must(err)
	return f
}

func urlJoin(s1, s2 string) string {
	if strings.HasSuffix(s1, "/") {
		if strings.HasPrefix(s2, "/") {
			return s1 + s2[1:]
		}
		return s1 + s2
	}

	if strings.HasPrefix(s2, "/") {
		return s1 + s2
	}
	return s1 + "/" + s2
}

func dataToLines(d []byte) []string {
	s := string(d)
	return strings.Split(s, "\n")
}

func reverseStringSlice(a []string) {
	n := len(a) / 2
	for i := 0; i < n; i++ {
		a[i], a[n-i] = a[n-i], a[i]
	}
}

// turn "010 Defining a SetterGetter" to "Defining a SetterGetter"
func cleanTitle(s string) string {
	idx := strings.Index(s, " ")
	if idx == -1 {
		return s
	}
	if idx > 6 {
		return s
	}
	maybeNum := s[:idx]
	rest := s[idx+1:]
	_, err := strconv.Atoi(maybeNum)
	if err != nil {
		return s
	}
	return strings.TrimSpace(rest)
}

// removes empty lines from the beginning and end of the array
func trimEmptyLines(lines []string) []string {
	for len(lines) > 0 && len(lines[0]) == 0 {
		lines = lines[1:]
	}

	for len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}

	n := len(lines)
	res := make([]string, 0, n)
	prevWasEmpty := false
	for i := 0; i < n; i++ {
		l := lines[i]
		shouldAppend := l != "" || !prevWasEmpty
		prevWasEmpty = l == ""
		if shouldAppend {
			res = append(res, l)
		}
	}
	return res
}

func countStartChars(s string, c byte) int {
	for i := range s {
		if s[i] != c {
			return i
		}
	}
	return len(s)
}

// remove longest common space/tab prefix on non-empty lines
func shiftLines(lines []string) {
	maxTabPrefix := 1024
	maxSpacePrefix := 1024
	// first determine how much we can remove
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		n := countStartChars(line, ' ')
		if n > 0 {
			if n < maxSpacePrefix {
				maxSpacePrefix = n
			}
			continue
		}
		n = countStartChars(line, '\t')
		if n > 0 {
			if n < maxTabPrefix {
				maxTabPrefix = n
			}
			continue
		}
		// if doesn't start with space or tab, early abort
		return
	}
	if maxSpacePrefix == 1024 && maxTabPrefix == 1024 {
		return
	}

	toRemove := maxSpacePrefix
	if maxTabPrefix != 1024 {
		toRemove = maxTabPrefix
	}
	if toRemove == 0 {
		return
	}

	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		lines[i] = line[toRemove:]
	}
}

var (
	softErrorMode bool
	delayedErrors []string
)

func maybePanicIfErr(err error) {
	if err == nil {
		return
	}
	if !softErrorMode {
		must(err)
	}
	delayedErrors = append(delayedErrors, err.Error())
}

func createDirForFileMaybeMust(path string) {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	maybePanicIfErr(err)
}

func copyFileMaybeMust(dst, src string) error {
	createDirForFileMaybeMust(dst)
	err := u.CopyFile(dst, src)
	maybePanicIfErr(err)
	return err
}

var (
	didBuildFrontEnd = false
)

func buildFrontend() {
	// only needs to do it once
	if didBuildFrontEnd {
		return
	}
	{
		os.Remove("package-lock.json")
		os.RemoveAll("node_modules")
		cmd := exec.Command("yarn", "install")
		runCmdMust(cmd)
	}
	// could also be
	// .\node_modules\.bin\rollup -c
	{
		cmd := exec.Command("yarn", "build-dev")
		runCmdMust(cmd)
	}
	didBuildFrontEnd = true
}

func fileExists(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.Mode().IsRegular()
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func dirExists(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.IsDir()
}

func fmtCmdShort(cmd exec.Cmd) string {
	cmd.Path = filepath.Base(cmd.Path)
	return cmd.String()
}

func runCmdMust(cmd *exec.Cmd) string {
	fmt.Printf("> %s\n", fmtCmdShort(*cmd))
	canCapture := (cmd.Stdout == nil) && (cmd.Stderr == nil)
	if canCapture {
		out, err := cmd.CombinedOutput()
		if err == nil {
			if len(out) > 0 {
				fmt.Printf("Output:\n%s\n", string(out))
			}
			return string(out)
		}
		fmt.Printf("cmd '%s' failed with '%s'. Output:\n%s\n", cmd, err, string(out))
		must(err)
		return string(out)
	}
	err := cmd.Run()
	if err == nil {
		return ""
	}
	fmt.Printf("cmd '%s' failed with '%s'\n", cmd, err)
	must(err)
	return ""
}

func readFileMust(path string) []byte {
	d, err := ioutil.ReadFile(path)
	must(err)
	return d
}

func writeFileMust(path string, data []byte) {
	err := ioutil.WriteFile(path, data, 0644)
	must(err)
}

func createDirForFile(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}

func formatSize(n int64) string {
	sizes := []int64{1024 * 1024 * 1024, 1024 * 1024, 1024}
	suffixes := []string{"GB", "MB", "kB"}
	for i, size := range sizes {
		if n >= size {
			s := fmt.Sprintf("%.2f", float64(n)/float64(size))
			return strings.TrimSuffix(s, ".00") + " " + suffixes[i]
		}
	}
	return fmt.Sprintf("%d bytes", n)
}

func getFileSize(path string) int64 {
	st, err := os.Lstat(path)
	if err == nil {
		return st.Size()
	}
	return -1
}

func isWindows() bool {
	return strings.Contains(runtime.GOOS, "windows")
}

// from https://gist.github.com/hyg/9c4afcd91fe24316cbf0
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func waitForCtrlC() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt /* SIGINT */, syscall.SIGTERM)
	<-c
}

func sha1HexOfBytes(data []byte) string {
	return fmt.Sprintf("%x", sha1OfBytes(data))
}

func sha1OfBytes(data []byte) []byte {
	h := sha1.New()
	h.Write(data)
	return h.Sum(nil)
}

func createDirMust(dir string) string {
	err := os.MkdirAll(dir, 0755)
	must(err)
	return dir
}
