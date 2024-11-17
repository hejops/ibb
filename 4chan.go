//

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Post struct {
	Board    string // must be inherited from parent Thread/Catalog
	Subject  string `json:"sub"` // often empty in Thread
	Comment  string `json:"com"` // raw html
	Filename string // original name at upload time
	Ext      string // starts with "."
	Time     int    `json:"tim"`
	Num      int    `json:"no"`
	// LastModified int `json:"last_modified"` // may be 0
}

// Render comment as HTML, then add quoted post(s) with indentation.
func (p Post) QuoteComment(t *Thread) string {
	// comm := renderHTML(p.Comment)
	var lines []string
	for _, line := range renderHTML(p.Comment) {
		lines = append(lines, line)
		if strings.HasPrefix(line, ">>") {
			quote := strings.Fields(line)[0]
			quote = strings.TrimPrefix(quote, ">>")
			id, err := strconv.Atoi(quote)
			if err != nil {
				panic(err)
			}
			parent, err := t.getIndex(id)
			if err != nil {
				continue
			}
			lines = append(lines, indent(t.Posts[parent].htmlComment())...)
		}
	}
	return strings.Join(lines, "\n")
}

func (p Post) htmlComment() []string {
	return renderHTML(p.Comment)
}

func (p Post) lineComment() (c string) {
	if p.Comment == "" {
		comment, _ := p.imageUrl()
		return "[" + comment + "]"
	}
	c = html.UnescapeString(p.Comment)
	c = stripHtmlTags(c)
	c = strings.ReplaceAll(c, "\n", " ")
	return c
}

func (p Post) imageUrl() (url string, err error) {
	if p.Time == 0 {
		return "", errors.New("no image")
	}

	if p.Board == "" {
		panic("empty board")
	}

	url = fmt.Sprintf("https://i.4cdn.org/%s/%d%s", p.Board, p.Time, p.Ext)
	return url, nil
}

// Returns path to temp image file
func (p Post) imagePath() (fname string, err error) {
	url, err := p.imageUrl()
	if err != nil {
		return "", err
	}

	path := filepath.Join(tmpDir, filepath.Base(url))
	return path, nil
}

func (p Post) download() {
	url, err := p.imageUrl()
	if err != nil {
		return
	}
	path, err := p.imagePath()
	if err != nil {
		return
	}

	if _, err := os.Stat(path); err == nil {
		return
	}

	log.Println("downloading", url)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	// _ = os.Mkdir(tmpDir, os.ModePerm)

	if err := os.WriteFile(path, b, 0666); err != nil {
		panic(err)
	}
}

type Catalog struct {
	Board string
	Posts []*Post // OPs
}

func getCatalog(board string) Catalog {
	url := fmt.Sprintf("https://a.4cdn.org/%s/catalog.json", board)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var pages []struct {
		Page    int
		Threads []*Post
	}
	if err := json.Unmarshal(b, &pages); err != nil {
		panic(err)
	}
	// fmt.Println(pages)

	var threads []*Post
	for _, p := range pages {
		threads = append(threads, p.Threads...)
	}

	// ensure all posts have Board field set (otherwise leads to erroneous
	// image urls)
	for _, p := range threads {
		p.Board = board
	}

	return Catalog{Board: board, Posts: threads}
}

// Get thread by subject
func (c Catalog) findThread(subject string) *Thread {
	var found *Post
	for _, t := range c.Posts {
		if strings.ToLower(t.Subject) == subject {
			found = t
			break
		}
	}

	if found == nil {
		return nil
	}

	// return c.getThread(found.Num)
	return getThread(c.Board, found.Num)
}

// Note that Thread has the same structure as Catalog, but lacks access to the
// findThread method
type Thread struct {
	Board string
	Posts []*Post
	// pointer because we need to mutate Post.Board
}

func (t *Thread) getIndex(id int) (int, error) {
	for i, p := range t.Posts {
		if p.Num == id {
			return i, nil
		}
	}
	return 0, errors.New("post not found")
}

func (t *Thread) filterPosts(s string) (matches []int) {
	for idx, p := range t.Posts {
		if strings.Contains(strings.ToLower(p.Comment+p.Subject), s) {
			matches = append(matches, idx)
		}
	}
	return matches
}

func (t *Thread) cleanImages() {
	for _, p := range t.Posts {
		path, err := p.imagePath()
		if err != nil {
			continue
		}
		_ = os.Remove(path)
	}
}

// Get thread by id
func getThread(board string, id int) *Thread {
	url := fmt.Sprintf("https://a.4cdn.org/%s/thread/%d.json", board, id)
	// log.Println("getting", url)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var t Thread
	if err := json.Unmarshal(b, &t); err != nil {
		panic(err)
	}
	t.Board = board
	// log.Printf(`getThread("%s", %d)`, board, id)

	for _, p := range t.Posts {
		p.Board = t.Board
	}
	return &t
}
