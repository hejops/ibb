package main

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

func kastden(user string) []string {
	url := "https://selca.kastden.org/owner/" + user // /?max_time=2024-08-25T00:00"
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		panic(err)
	}

	// fmt.Println(filter([]int{1, 0, 3, 0, 5}, 0))

	// </div><div data-media_id="xxxxxxx" data-type="image" data-added_at="xxxx-xx-xx xx:xx:xx.xxxxxx" data-created_at="xxxx-xx-xx xx:xx:xx" data-timestamp="" data-post_id="xxxxxxxx" data-account_id="xxxx" data-idx="x" data-owner_display_name="abcd" data-filename="xxxxxxxxx_xxxxxxxxxxxxxxx_xxxxxxxxxxxxxxxxxxx_n.heic.jpg" data-favorite_count="x" data-tag_count="x" class="entry">

	// https://pkg.go.dev/github.com/PuerkitoBio/goquery#pkg-overview

	// feed is sorted by post chronologically descending, but within each
	// post, images are sorted ascending. so the ids are something like:
	// 10, 11, 12, 5, 6, 7, ...

	// fmt.Println(doc.First().Html())
	matches := doc.Find("div").Map(func(i int, s *goquery.Selection) string {
		filename, _ := s.Attr("data-filename") //.Html()
		if filename == "" {
			return ""
		}
		// apparently these 2 fields are enough to generate valid img urls
		id, _ := s.Attr("data-media_id") //.Html()
		return fmt.Sprintf("https://selca.kastden.org/original/%s/%s", id, filename)
	})

	return filter(matches, "")
}
