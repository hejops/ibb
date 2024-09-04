package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Catalog struct {
	Board string
	Posts []*Post // OPs
}

type Thread struct {
	Board string
	Posts []*Post
	// pointer because we need to mutate Post.Board
}

type Post struct {
	Board    string
	Subject  string `json:"sub"` // often empty
	Comment  string `json:"com"`
	Filename string
	Ext      string
	Time     int `json:"tim"`
	Num      int `json:"no"`
	// LastModified int `json:"last_modified"` // may be 0
}

func (p Post) lineComment() (c string) {
	c = html.UnescapeString(p.Comment)
	c = stripHtmlTags(c)
	return c
}

func (p Post) imageUrl() (string, error) {
	if p.Time == 0 {
		return "", errors.New("no image")
	}
	return fmt.Sprintf("https://i.4cdn.org/%s/%d%s", p.Board, p.Time, p.Ext), nil
}

// Returns empty string if current post has no image
func (p Post) download() (string, error) {
	url, err := p.imageUrl()
	if err != nil {
		return "", err
	}

	_ = os.Mkdir("out", os.ModePerm)
	base := filepath.Base(url)
	path := filepath.Join("out", base)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

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

	return Catalog{Board: board, Posts: threads}
}

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

	url := fmt.Sprintf("https://a.4cdn.org/hr/thread/%d.json", found.Num)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	// var t Thread
	t := Thread{Board: c.Board}
	if err := json.Unmarshal(b, &t); err != nil {
		panic(err)
	}

	for _, p := range t.Posts {
		p.Board = t.Board
	}
	return &t
}
