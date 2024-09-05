// Imageboard browser
//
// Primarily supports 4chan, with kastden support planned

package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const tmpDir = "/tmp/ibb"

func main() {
	// urls := kastden(user)

	lf, _ := tea.LogToFile("/tmp/ibb.log", "ibb")
	defer lf.Close()

	log.Println("started")

	var p *tea.Program

	switch len(os.Args) {
	case 1:
		// select board

	case 2:
		// TODO: on g, first render of catalog is always erroneous
		// (overlap), even when returning from a thread (img not
		// padded). on the other hand, threads always render correctly!
		// but on hr, catalog is fine, which suggests the error is
		// specific to that rms image (lol)
		board := os.Args[1]
		c := getCatalog(board)
		t := Thread(c)
		// log.Println(c.Board, t.Board)
		p = tea.NewProgram(
			&ThreadViewer{thread: t, catalog: true},
			tea.WithAltScreen(),
		)

	case 3:
		board, subject := os.Args[1], os.Args[2]
		t := getCatalog(board).findThread(subject)
		p = tea.NewProgram(
			&ThreadViewer{thread: *t, catalog: false},
			tea.WithAltScreen(),
		)

	default:
		panic(1)
	}

	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
