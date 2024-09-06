//

package main

import (
	"bytes"
	"os/exec"
)

// const ScrollOff = 10

// Render html with w3m; note that w3m appends a newline
func renderHTML(s string) string {
	cmd := exec.Command("w3m", "-dump", "-T", "text/html")

	// https://blog.kowalczyk.info/article/wOYk/advanced-command-execution-in-go-with-osexec.html#behind-the-scenes-of-combinedoutput
	var b bytes.Buffer
	cmd.Stdout = &b

	in, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	if _, err := in.Write([]byte(s)); err != nil {
		panic(err)
	}
	if err := in.Close(); err != nil {
		panic(err)
	}

	if err := cmd.Run(); err != nil {
		panic(err)
	}

	return b.String()
}

func stripHtmlTags(s string) string {
	var count int
	var inner []rune
	var chars []rune
	for _, c := range s {
		// fmt.Println(c, string(inner))
		switch {
		case c == '<':
			count++
			continue
		case c == '>' && count > 0:
			count--
			if count == 0 && string(inner) == "br" {
				chars = append(chars, ' ')
			}
			inner = []rune{}
			continue
		case count > 0:
			inner = append(inner, c)
			continue
		default:
			chars = append(chars, c)
		}
	}
	return string(chars)
}

// Returns [start,end) range
func getScrollWindow[T any](
	current int,
	slice *[]T,
	// limit int, // very prone to off-by-1 footguns
	scrolloff int,
) (start int, end int) {
	limit := len(*slice) - 1
	switch {
	case current <= scrolloff:
		start, end = 0, min(limit, scrolloff*2)
	case limit-current <= scrolloff:
		start, end = max(0, limit-2*scrolloff), limit
	default:
		start, end = current-scrolloff, current+scrolloff
	}
	return start, end + 1
}

func filter[T comparable](arr []T, remove T) []T {
	i := 0
	for _, x := range arr {
		if x == remove {
			continue
		}
		arr[i] = x
		i++
	}
	return arr[:i]
}
