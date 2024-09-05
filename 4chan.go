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
	"strings"
)

type Post struct {
	Board    string
	Subject  string `json:"sub"` // often empty in Thread
	Comment  string `json:"com"`
	Filename string // original name at upload time
	Ext      string // starts with "."
	Time     int    `json:"tim"`
	Num      int    `json:"no"`
	// LastModified int `json:"last_modified"` // may be 0
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

// Returns path to temp image file, or empty string if current post has no
// image
func (p Post) download() (string, error) {
	url, err := p.imageUrl()
	if err != nil {
		return "", err
	}

	_ = os.Mkdir(tmpDir, os.ModePerm)
	path := filepath.Join(tmpDir, filepath.Base(url))
	if _, err := os.Stat(path); err == nil {
		return path, nil
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
	if err := os.WriteFile(path, b, 0666); err != nil {
		panic(err)
	}
	return path, nil
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

func (t *Thread) filterPosts(s string) (matches []int) {
	for idx, p := range t.Posts {
		if strings.Contains(strings.ToLower(p.Comment+p.Subject), s) {
			matches = append(matches, idx)
		}
	}
	return matches
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
