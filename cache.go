package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/kjk/siser"
	"github.com/kjk/u"
)

const (
	recNameGist       = "gist"       // content of the gist
	recNameGistOutput = "gistoutput" // result of evaluating the gist
)

// CacheGist represents a cached gist
type CacheGist struct {
	ID   string
	Gist string
}

// CacheGistOutput represents a cached output of evaluating a gist
type CacheGistOutput struct {
	Gist   string
	Output string
	Sha1   string // sha1 of Gist
}

// Cache represents a cache
type Cache struct {
	// path of the cache file
	path        string
	gists       []*CacheGist
	gistOutputs []*CacheGistOutput
}

func recGetMust(rec *siser.Record, name string) string {
	v, ok := rec.Get(name)
	u.PanicIf(!ok)
	return v
}

func recGetMustNonEmpty(rec *siser.Record, name string) string {
	v, ok := rec.Get(name)
	u.PanicIf(!ok || v == "")
	return v
}

func (c *Cache) saveRecord(rec *siser.Record) error {
	f := openForAppend(c.path)
	defer u.CloseNoError(f)
	w := siser.NewWriter(f)
	_, err := w.WriteRecord(rec)
	return err
}

// return true if gist changed
func (c *Cache) saveGist(gistID, gistContent string) bool {
	rec := &siser.Record{
		Name: recNameGist,
	}
	existing := c.getGistByID(gistID)
	if existing != nil && existing.Gist == gistContent {
		return false
	}
	u.PanicIf(gistID == "" || gistContent == "")
	rec.Write("GistID", gistID)
	rec.Write("Gist", gistContent)
	err := c.saveRecord(rec)
	must(err)

	gist := &CacheGist{
		ID:   gistID,
		Gist: gistContent,
	}
	c.gists = append(c.gists, gist)
	return true
}

func (c *Cache) loadGist(rec *siser.Record) {
	u.PanicIf(rec.Name != recNameGist)
	gist := &CacheGist{
		ID:   recGetMustNonEmpty(rec, "GistID"),
		Gist: recGetMustNonEmpty(rec, "Gist"),
	}
	c.gists = append(c.gists, gist)
}

func (c *Cache) getGistByID(id string) *CacheGist {
	n := len(c.gists)
	// read from the end because newer entries are at the end
	// and over-write older entries
	for i := n - 1; i >= 0; i-- {
		r := c.gists[i]
		if r.ID == id {
			return r
		}
	}
	return nil
}

func (c *Cache) saveGistOutput(gist, output string) {
	rec := &siser.Record{
		Name: recNameGistOutput,
	}
	//TODO: probably remove, it's ok to have no output
	//u.PanicIf(output == "")
	rec.Write("Gist", gist)
	rec.Write("GistOutput", output)
	err := c.saveRecord(rec)
	must(err)
	sha1 := u.Sha1HexOfBytes([]byte(gist))
	gout := &CacheGistOutput{
		Gist:   gist,
		Output: output,
		Sha1:   sha1,
	}
	c.gistOutputs = append(c.gistOutputs, gout)
}

func (c *Cache) loadGistOutput(rec *siser.Record) {
	gist := recGetMustNonEmpty(rec, "Gist")
	output := recGetMust(rec, "GistOutput")
	sha1 := u.Sha1HexOfBytes([]byte(gist))
	gout := &CacheGistOutput{
		Gist:   gist,
		Output: output,
		Sha1:   sha1,
	}
	c.gistOutputs = append(c.gistOutputs, gout)
}

func (c *Cache) getGistOuputBySha1(sha1 string) *CacheGistOutput {
	n := len(c.gistOutputs)
	// read from the end because newer entries are at the end
	// and over-write older entries
	for i := n - 1; i >= 0; i-- {
		r := c.gistOutputs[i]
		if sha1 == r.Sha1 {
			return r
		}
	}
	return nil
}

func loadCache(book *Book) *Cache {
	path := book.cachePath()
	logf("loadCache: %s\n", path)
	u.CreateDirForFileMust(path)

	c := &Cache{
		path: path,
	}

	f, err := os.Open(path)
	if err != nil {
		// it's ok if file doesn't exist
		logf("  cache file %s doesn't exist\n", path)
		return c
	}
	defer u.CloseNoError(f)

	nRecords := 0
	r := siser.NewReader(bufio.NewReader(f))
	for r.ReadNextRecord() {
		rec := r.Record
		switch rec.Name {
		case recNameGist:
			c.loadGist(rec)
		case recNameGistOutput:
			c.loadGistOutput(rec)
		default:
			panic(fmt.Errorf("unknown record type: '%s'", rec.Name))
		}
		nRecords++
	}
	must(r.Err())
	logf(" got %d cache records\n", nRecords)
	return c
}
