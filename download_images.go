package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/u"

	"github.com/kjk/notionapi"
)

func guessExt(fileName string, contentType string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	// TODO: maybe allow every non-empty extension. This
	// white-listing might not be a good idea
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff", ".svg":
		return ext
	}

	contentType = strings.ToLower(contentType)
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/svg+xml":
		return ".svg"
	}
	panic(fmt.Errorf("didn't find ext for file '%s', content type '%s'", fileName, contentType))
}

func downloadImage(c *notionapi.Client, parentTable string, blockID string, uri string) ([]byte, string, error) {
	img, err := c.DownloadFile(uri, blockID, "")
	if err != nil {
		return nil, "", err
	}
	ext := guessExt(uri, img.Header.Get("Content-Type"))
	return img.Data, ext, nil
}

func sha1OfLink(link string) string {
	link = strings.ToLower(link)
	h := sha1.New()
	h.Write([]byte(link))
	return fmt.Sprintf("%x", h.Sum(nil))
}

var (
	dirToImgFiles = map[string][]os.FileInfo{}
)

// checks if an image corresponding to this sha1 is present
// (same name after removing extension)
func findImageInDir(imgDir string, sha1 string) string {
	imgFiles := dirToImgFiles[imgDir]
	if imgFiles == nil {
		imgFiles, _ = ioutil.ReadDir(imgDir)
		dirToImgFiles[imgDir] = imgFiles
	}
	for _, fi := range imgFiles {
		if strings.HasPrefix(fi.Name(), sha1) {
			return filepath.Join(imgDir, fi.Name())
		}
	}
	return ""
}

// return path of cached image on disk
func downloadAndCacheImage(c *notionapi.Client, block *notionapi.Block, imgDir string, uri string) (string, error) {
	err := os.MkdirAll(imgDir, 0755)
	must(err)

	sha := sha1OfLink(uri)
	cachedPath := findImageInDir(imgDir, sha)
	if cachedPath != "" {
		logVerbose("Image %s already downloaded as %s\n", uri, cachedPath)
		return cachedPath, nil
	}

	timeStart := time.Now()
	logf("Downloading %s ... ", uri)
	imgData, ext, err := downloadImage(c, block.ParentTable, block.ID, uri)
	if err != nil {
		logf("\n  Failed with %s\n", err)
		return "", err
	}

	cachedPath = filepath.Join(imgDir, sha+ext)

	u.WriteFileMust(cachedPath, imgData)
	logf(" in %s.\nWrote as '%s'\n", time.Since(timeStart), cachedPath)

	return cachedPath, nil
}

func downloadAndRememberImage(page *Page, block *notionapi.Block, imgDir string, link string) {
	if !isFullURL(link) {
		id := toNoDashID(page.NotionID)
		logf("downloadAndRememberImage(): skipping '%s' because not a valid url\npage: https://notion.so/%s\n\n", link, id)
		return
	}

	client := newNotionClient()
	path, err := downloadAndCacheImage(client, block, imgDir, link)
	if err != nil {
		id := toNoDashID(page.NotionID)
		logf("downloadAndCacheImage('%s') from page https://notion.so/%s failed with '%s'\n", link, id, err)
		must(err)
	}
	relURL := "/img/" + filepath.Base(path)
	im := &ImageMapping{
		link:        link,
		path:        path,
		relativeURL: relURL,
	}
	page.Images = append(page.Images, im)
}

func downloadImages(book *Book, page *Page) {
	if flgNoDownload {
		return
	}
	imgDir := filepath.Join(book.NotionCacheDir, "img")

	handleImage := func(block *notionapi.Block) {
		downloadAndRememberImage(page, block, imgDir, block.Source)
	}

	fn := func(block *notionapi.Block) {
		switch block.Type {
		case notionapi.BlockImage:
			handleImage(block)
		}
	}
	root := page.NotionPage.Root()
	format := root.FormatPage()
	if format.PageCover != "" {
		downloadAndRememberImage(page, root, imgDir, format.PageCover)
	}
	page.NotionPage.ForEachBlock(fn)
}
