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

	"github.com/mattn/go-sixel"
	"github.com/nfnt/resize"
)

type Size struct {
	width  int // in chars
	height int // in chars
}

// we assume 1 char is 15 px
const PixPerChar = 15

func decode(fname string) (img image.Image) {
	fo, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer fo.Close()

	var ierr error
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
	if ierr != nil {
		panic(err)
	}
	return img
}

// Render image to stdout. Constraining the image to a given size is
// recommended, as rendering time scales quadratically with image size.
func Render(fname string, size *Size) {
	img := decode(fname)
	// img size in pixels
	imgX := img.Bounds().Max.X
	imgY := img.Bounds().Max.Y

	log.Println("img dims:", imgX, imgY)

	if size != nil {
		// https://en.wikipedia.org/wiki/Comparison_gallery_of_image_scaling_algorithms
		interp := resize.Lanczos2 // Bilinear, Bicubic

		// not sure why this works but ok
		target := uint(float64(size.width*PixPerChar) / 4)

		switch imgY > imgX {
		case true: // tall -- resize by width
			img = resize.Resize(target, 0, img, interp)
		case false: // wide -- resize by height
			img = resize.Resize(0, target, img, interp)
		}

		imgX = img.Bounds().Max.X
		imgY = img.Bounds().Max.Y

		if imgX > 1200 || imgY > 1200 {
			panic(fmt.Sprintln("too big:", imgX, imgY))
		}

		log.Println("resized dims:", imgX, imgY)

		xPad := (size.width - (imgX / PixPerChar)) / 2
		yPad := (size.height-(imgY/PixPerChar))/2 - 10
		// log.Println("yPad", yPad)

		fmt.Println(strings.Repeat("\n", max(0, yPad)))
		fmt.Println(strings.Repeat(" ", xPad))

	}

	if err := sixel.NewEncoder(os.Stdout).Encode(img); err != nil {
		panic(err)
	}
}
