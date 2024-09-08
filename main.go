// Imageboard browser
//
// Primarily supports 4chan, with kastden support planned (once I figure out
// how to render a grid of images)

package main

import (
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	tea "github.com/charmbracelet/bubbletea"
)

const tmpDir = "/tmp/ibb"

func main() {
	// TODO: find memory leak (probably not .Close-ing something somewhere)

	// urls := kastden(user)

	lf, _ := tea.LogToFile("/tmp/ibb.log", "ibb")
	defer lf.Close()

	log.Println("started")

	var p *tea.Program

	switch len(os.Args) {
	case 1:

		tv()

		return

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

	// https://github.com/google/pprof/blob/main/doc/README.md#flame-graph
	// https://go.dev/doc/diagnostics
	// https://stackoverflow.com/questions/24863164/how-to-analyze-golang-memory
	// https://www.brendangregg.com/FlameGraphs/memoryflamegraphs.html

	// echo web | go tool pprof *.prof

	// https://pkg.go.dev/runtime/pprof#hdr-Profiling_a_Go_program
	f, _ := os.Create("mem.prof")
	defer f.Close()
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
}
