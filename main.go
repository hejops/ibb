// Imageboard browser
//
// Primarily supports 4chan, with kastden support planned

package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// urls := kastden(user)

	lf, _ := tea.LogToFile("/tmp/img.log", "img")
	defer lf.Close()

	var p *tea.Program

	log.Println(len(os.Args))
	switch len(os.Args) {
	case 1:
		// select board

	// case 2:
	// 	// select thread
	// 	board := os.Args[1]
	// 	c := getCatalog(board)
	// 	p = tea.NewProgram(
	// 		&CatalogViewer{catalog: c},
	// 		tea.WithAltScreen(),
	// 	)

	case 3:
		board, subject := os.Args[1], os.Args[2]
		t := getCatalog(board).findThread(subject)
		p = tea.NewProgram(
			&ThreadViewer{thread: *t},
			tea.WithAltScreen(),
		)

	default:
		panic(1)
	}

	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
