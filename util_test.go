package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtil(t *testing.T) {
	type test struct {
		c     int
		lim   int
		start int
		end   int
	}

	ints := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	for _, x := range []test{
		{c: 0, lim: 9, start: 0, end: 5},
		{c: 1, lim: 9, start: 0, end: 5},
		{c: 2, lim: 9, start: 0, end: 5},

		{c: 3, lim: 9, start: 1, end: 6},
		{c: 4, lim: 9, start: 2, end: 7},
		{c: 5, lim: 9, start: 3, end: 8},
		{c: 6, lim: 9, start: 4, end: 9},

		{c: 7, lim: 9, start: 5, end: 10},
		{c: 8, lim: 9, start: 5, end: 10},
		{c: 9, lim: 9, start: 5, end: 10},
	} {
		start, end := getScrollWindow(x.c, &ints, 2)
		assert.Equal(t, start, x.start)
		assert.Equal(t, end, x.end)
	}

	for _, x := range []test{
		{c: 0, lim: 9, start: 0, end: 10},
		{c: 9, lim: 9, start: 0, end: 10},
	} {
		start, end := getScrollWindow(x.c, &ints, 15)
		assert.Equal(t, start, x.start)
		assert.Equal(t, end, x.end)
	}

	assert.Equal(t, stripHtmlTags("foo<wbr>bar"), "foobar")

	assert.Equal(
		t,
		stripHtmlTags(`<a href="#pxxxxxxx" class="quotelink">>>xxxxxxx</a><br>foo`),
		`>>xxxxxxx foo`,
	)
}
