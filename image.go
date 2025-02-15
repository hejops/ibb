// if performance is bad, consider wezterm imgcat <file> as a last resort

// https://github.com/gdamore/tcell/blob/88b9c25c3c5ee48b611dfeca9a2e9cf07812c35e/_demos/sixel.go#L39
// https://github.com/otiai10/gat/blob/228567838a2b2db10cc55d0e9bf74707967fa671/render/sixel.go

package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dolmen-go/kittyimg" // faster than mattn/go-sixel
	"github.com/nfnt/resize"
)

type Size struct {
	width  int // in chars
	height int // in chars
}

const (
	CharHeightPx = 15
	CharWidthPx  = 9
)

func decode(fname string) (img image.Image, ierr error) {
	fo, err := os.Open(fname)
	if err != nil {
		// panic(err)
		return nil, err
	}
	defer fo.Close()

	// var ierr error
	switch filepath.Ext(fname) {
	case ".png":
		img, ierr = png.Decode(fo)

	case ".jpg":
		img, ierr = jpeg.Decode(fo)

	// TODO: handle webm thumbnail?

	default:
		fmt.Println(fname)
		log.Println("failed to decode", fname)
		return
	}
	// if ierr != nil {
	// 	// panic(err)
	// 	return nil
	// }
	return img, ierr
}

// Render a single image to stdout. Constraining the image to a given size is
// recommended, as rendering time scales quadratically with image size.
func Render(fname string, size *Size) {
	img, err := decode(fname)
	if err != nil {
		return
	}

	if img == nil {
		return
	}
	// img size in pixels
	imgX := img.Bounds().Max.X
	imgY := img.Bounds().Max.Y

	if size != nil {

		// https://en.wikipedia.org/wiki/Comparison_gallery_of_image_scaling_algorithms
		interp := resize.Lanczos2 // Bilinear, Bicubic

		log.Println("img dims:", imgX, imgY)

		// always resize by height, preserving aspect ratio
		// note that size refers to the entire screen, so we /2 to
		// account for upper pane. this should eventually be the
		// responsibility of caller
		img = resize.Resize(
			0,
			uint(CharHeightPx*55), // 55 is magic, apparently
			img,
			interp,
		)

		imgX = img.Bounds().Max.X
		imgY = img.Bounds().Max.Y

		log.Println("resized dims:", imgX, imgY)

		xPad := (size.width - (imgX / CharHeightPx)) / 2
		yPad := (size.height - (imgY / CharWidthPx)) / 2
		// log.Println("yPad", yPad)

		fmt.Println(strings.Repeat("\n", max(0, yPad)))
		fmt.Println(strings.Repeat(" ", xPad))

	}

	// if err := sixel.NewEncoder(os.Stdout).Encode(img); err != nil {
	// 	panic(err)
	// }

	if err := kittyimg.Fprintln(os.Stdout, img); err != nil {
		panic(err)
	}

	// the result is a b64-encoded string; s and v refer to width and
	// height, for example. the protocol spec is 'loosely' documented
	// within kitty, but more concrete examples can be gleaned from related
	// repos.
	//
	// https://sw.kovidgoyal.net/kitty/graphics-protocol/#the-graphics-escape-code
	// https://github.com/benjajaja/ratatui-image/blob/afbdd4e79251ef0709e4a2d9281b3ac6eb73291a/src/protocol/kitty.rs#L150
}
