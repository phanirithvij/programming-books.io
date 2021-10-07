package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/kjk/cheatsheets/pkg/filerotate"
	"github.com/kjk/siser"
)

var (
	logFile *os.File
)

var (
	logsDirCached = ""
	httpLogSiser  *siser.Writer
	httpLogRec    siser.Record
	httpLogMu     sync.Mutex
)

func getLogsDir() string {
	if logsDirCached != "" {
		return logsDirCached
	}
	logsDirCached = "logs"
	must(os.MkdirAll(logsDirCached, 0755))
	return logsDirCached
}

// <dir>/httplog-2021-10-06_01.txt.br
// =>
//apps/cheatsheet/httplog/2021/10-06/2021-10-06_01.txt.br
// return "" if <path> is in unexpected format
func remotePathFromFilePath(path string) string {
	name := filepath.Base(path)
	parts := strings.Split(name, "_")
	if len(parts) != 2 {
		return ""
	}
	// parts[1]: 01.txt.br
	hr := strings.Split(parts[1], ".")[0]
	if len(hr) != 2 {
		return ""
	}
	// parts[0]: httplog-2021-10-06
	parts = strings.Split(parts[0], "-")
	if len(parts) != 4 {
		return ""
	}
	year := parts[1]
	month := parts[2]
	day := parts[3]
	name = fmt.Sprintf("%s/%s-%s/%s-%s-%s_%s.txt.br", year, month, day, year, month, day, hr)
	return "apps/progbooks/httplog/" + name
}

// upload httplog-2021-10-06_01.txt as
// apps/cheatsheet/httplog/2021/10-06/2021-10-06_01.txt.br
func uploadCompressedHTTPLog(path string) {
	logf(ctx(), "uploadCompressedHTTPLog\n")
	pathBr := path + ".br"
	createCompressed := func() error {
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()
		os.Remove(pathBr)
		f, err := os.Create(pathBr)
		if err != nil {
			return err
		}
		w := brotli.NewWriterLevel(f, brotli.BestCompression)
		_, err = io.Copy(w, r)
		err2 := w.Close()
		err3 := f.Close()
		if err != nil {
			return err
		}
		if err2 != nil {
			return err2
		}
		return err3
	}
	defer os.Remove(pathBr)

	timeStart := time.Now()
	err := createCompressed()
	if err != nil {
		logf(ctx(), "uploadCompressedHTTPLog: createCompressed() failed with '%s'\n", err)
		return
	}
	dur := time.Since(timeStart)
	origSize := getFileSize(path)
	comprSize := getFileSize(pathBr)
	p := perc(origSize, comprSize)
	logf(ctx(), "uploadCompressedHTTPLog: compressed '%s' as '%s', %s => %s (%.2f%%) in %s\n", path, pathBr, formatSize(origSize), formatSize(comprSize), p, dur)

	timeStart = time.Now()
	mc := newMinioSpacesClient()
	remotePath := remotePathFromFilePath(pathBr)
	if remotePath == "" {
		logf(ctx(), "uploadCompressedHTTPLog: remotePathFromFilePath() failed for '%s'\n", pathBr)
		return
	}
	err = minioUploadFilePublic(mc, remotePath, pathBr)
	if err != nil {
		logerrf(ctx(), "uploadCompressedHTTPLog: minioUploadFilePublic() failed with '%s'\n", err)
		return
	}
	logf(ctx(), "uploadCompressedHTTPLog: uploaded '%s' as '%s' in %s\n", pathBr, remotePath, time.Since(timeStart))
}

func didRotateHTTPLog(path string, didRotate bool) {
	canUpload := hasSpacesCreds()
	logf(ctx(), "didRotateHTTPLog: '%s', didRotate: %v, hasSpacesCreds: %v\n", path, didRotate, canUpload)
	if !canUpload || !didRotate {
		return
	}
	go uploadCompressedHTTPLog(path)
}

func NewLogHourly(dir string, didClose func(path string, didRotate bool)) (*filerotate.File, error) {
	hourly := func(creationTime time.Time, now time.Time) string {
		if filerotate.IsSameHour(creationTime, now) {
			return ""
		}
		name := "httplog-" + now.Format("2006-01-02_15") + ".txt"
		path := filepath.Join(dir, name)
		logf(ctx(), "NewLogHourly: '%s'\n", path)
		return path
	}
	config := filerotate.Config{
		DidClose:           didClose,
		PathIfShouldRotate: hourly,
	}
	return filerotate.New(&config)
}

func openHTTPLog() func() {
	dir := getLogsDir()

	logFile, err := NewLogHourly(dir, didRotateHTTPLog)
	must(err)
	httpLogSiser = siser.NewWriter(logFile)
	// TODO: should I change filerotate so that it opens the file immedaitely?
	logf(context.Background(), "opened http log file '%s'\n", logFile.Path)
	return func() {
		_ = logFile.Close()
		httpLogSiser = nil
	}
}

func fmtSmart(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

func openLog() func() {
	var err error
	logFile, err = os.Create("log.txt")
	must(err)
	return func() {
		_ = logFile.Close()
		logFile = nil
	}
}

func logsf(format string, args ...interface{}) {
	s := fmtSmart(format, args...)
	fmt.Print(s)
	if logFile != nil {
		_, _ = fmt.Fprint(logFile, s)
	}
}

func logerrf(ctx context.Context, format string, args ...interface{}) {
	s := format
	if len(args) > 0 {
		s = fmt.Sprintf(format, args...)
	}
	fmt.Printf("Error: %s", s)
}

func logf(ctx context.Context, format string, args ...interface{}) {
	s := fmtSmart(format, args...)
	fmt.Print(s)
	if logFile != nil {
		_, _ = fmt.Fprint(logFile, s)
	}
}

func logvf(format string, args ...interface{}) {
	if logFile == nil {
		return
	}
	s := fmtSmart(format, args...)
	_, _ = fmt.Fprint(logFile, s)
}

var (
	hdrsToNotLog = []string{
		"Connection",
		"Sec-Ch-Ua-Mobile",
		"Sec-Fetch-Dest",
		"Sec-Ch-Ua-Platform",
		"Dnt",
		"Upgrade-Insecure-Requests",
		"Sec-Fetch-Site",
		"Sec-Fetch-Mode",
		"Sec-Fetch-User",
		"If-Modified-Since",
		"Accept-Language",
		"Cf-Ray",
		"CF-Visitor",
		"X-Request-Start",
		"Cdn-Loop",
		"X-Forwarded-Proto",
	}
	hdrsToNotLogMap map[string]bool
)

func shouldLogHeader(s string) bool {
	if hdrsToNotLogMap == nil {
		hdrsToNotLogMap = map[string]bool{}
		for _, h := range hdrsToNotLog {
			h = strings.ToLower(h)
			hdrsToNotLogMap[h] = true
		}
	}
	s = strings.ToLower(s)
	return !hdrsToNotLogMap[s]
}

func recWriteNonEmpty(rec *siser.Record, k, v string) {
	if v != "" {
		rec.Write(k, v)
	}
}

func logHTTPReq(r *http.Request, code int, size int64, dur time.Duration) {
	uri := r.URL.Path
	if !strings.HasPrefix(uri, "/ping") {
		logf(ctx(), "%s %s %d in %s\n", r.Method, r.RequestURI, code, dur)
	}

	shouldLogURL := func() bool {
		// we don't want to do deatiled logging for all files, to make
		// the log files smaller
		ext := strings.ToLower(filepath.Ext(uri))
		switch ext {
		case ".css", ".js", ".ico":
			return false
		}
		if strings.HasPrefix(uri, "/ping") {
			// our internal health monitoring endpoint
			return false
		}
		return true
	}
	if !shouldLogURL() {
		return
	}

	httpLogMu.Lock()
	defer httpLogMu.Unlock()

	if httpLogSiser == nil {
		return
	}

	rec := &httpLogRec
	rec.Reset()
	rec.Write("req", fmt.Sprintf("%s %s %d", r.Method, r.RequestURI, code))
	recWriteNonEmpty(rec, "host", r.Host)
	rec.Write("ipaddr", requestGetRemoteAddress(r))
	rec.Write("size", strconv.FormatInt(size, 10))
	durMicro := int64(dur / time.Microsecond)
	rec.Write("durmicro", strconv.FormatInt(durMicro, 10))

	for k, v := range r.Header {
		if !shouldLogHeader(k) {
			continue
		}
		if len(v) > 0 && len(v[0]) > 0 {
			rec.Write(k, v[0])
		}
	}

	_, err := httpLogSiser.WriteRecord(rec)
	if err != nil {
		logerrf(ctx(), "logHTTPReq: httpLogSiser.WriteRecord() failed with '%s'\n", err)
	}
}

// requestGetRemoteAddress returns ip address of the client making the request,
// taking into account http proxies
func requestGetRemoteAddress(r *http.Request) string {
	hdr := r.Header
	hdrRealIP := hdr.Get("x-real-ip")
	hdrForwardedFor := hdr.Get("x-forwarded-for")
	// Request.RemoteAddress contains port, which we want to remove i.e.:
	// "[::1]:58292" => "[::1]"
	ipAddrFromRemoteAddr := func(s string) string {
		idx := strings.LastIndex(s, ":")
		if idx == -1 {
			return s
		}
		return s[:idx]
	}
	if hdrRealIP == "" && hdrForwardedFor == "" {
		return ipAddrFromRemoteAddr(r.RemoteAddr)
	}
	if hdrForwardedFor != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(hdrForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		// TODO: should return first non-local address
		return parts[0]
	}
	return hdrRealIP
}
