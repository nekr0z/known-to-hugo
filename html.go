// Copyright (C) 2020 Evgeny Kuznetsov (evgeny@kuznetsov.md)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
)

type page interface {
	title() string
	date() time.Time
	content() pageContent
	canonicalUrl() string
	tags() []string
	webmentions() []byte
}

type pageContent struct {
	*goquery.Selection
}

type comment interface {
	author() author
	content() content
	url() string
	date() string
}

func blogDir(input, output, blogType string) {
	_ = filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("%s: %v\n", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		s, err := loadHtmlFile(filepath.Join(path))
		if err != nil {
			fmt.Printf("%s: %v\n", path, err)
			return nil
		}
		var p page
		switch blogType {
		case "diary":
			p = diaryPage{s}
		case "ljbackup":
			p = ljbPage{s}
		case "gplus":
			p = gpPage{s}
		default:
			fmt.Println("not implemented")
		}
		url := p.canonicalUrl()
		if (url == "" || url != filepath.Base(path)) && blogType != "gplus" {
			return nil
		}

		outPath := filepath.Join(output, strconv.Itoa(p.date().Year()), strings.TrimSuffix(url, filepath.Ext(path)))
		if err := os.MkdirAll(outPath, 0755); err != nil {
			panic(err)
		}

		cnt := p.content()
		images := cnt.processImages()
		downloadImages(outPath, images)

		outFile := filepath.Join(outPath, "index.md")

		b := hugo(p, draft)
		if err := ioutil.WriteFile(outFile, b, 0644); err != nil {
			fmt.Printf("%s: %v\n", outFile, err)
		}

		b = p.webmentions()
		if len(b) > 0 {
			outFile := filepath.Join(outPath, "comments.json")
			if err := ioutil.WriteFile(outFile, b, 0644); err != nil {
				fmt.Printf("%s: %v\n", outFile, err)
			}
		}

		return nil
	})
}

func hugo(p page, draft bool) []byte {
	b := getFM(p, draft)
	ct := p.content()
	b = append(b, ct.md()...)
	return b
}

func getFM(p page, draft bool) []byte {
	var frontMatter = map[string]interface{}{
		"title": p.title(),
		"date":  p.date(),
		"tags":  p.tags(),
		//		"reply_to":       getInReply(sel),
		//		"posse":          getSyndications(sel),
		//		"like_of":        getLikeOf(sel),
		"draft": draft,
	}
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(frontMatter); err != nil {
		panic(err)
	}
	var b []byte
	b = append(b, []byte(frontMatterSeparator)...)
	b = append(b, buf.Bytes()...)
	b = append(b, []byte(frontMatterSeparator)...)
	return b
}

func (c *pageContent) md() []byte {
	converter := md.NewConverter("", true, nil)
	got := converter.Convert(c.Unwrap())
	return []byte(got)
}

func (c *pageContent) processImages() map[string]string {
	se := c.Unwrap()
	out := map[string]string{}
	se.Find("img").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("src")
		fn := "image" + strconv.Itoa(i)

		// fix hrefs
		changeHrefs(se, link, fn)
		s.SetAttr("src", fn)

		out[fn] = link
	})
	return out
}

func getWebmention(cmt comment) mention {
	var m = mention{
		Type:   "entry",
		Author: cmt.author(),
	}
	m.Content = cmt.content()
	m.Property = "in-reply-to"
	m.Url = cmt.url()
	m.Date = cmt.date()
	return m
}

func loadHtmlFile(path string) (*goquery.Selection, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return nil, err
	}

	enc := getEncoding(doc.Find("html"))
	if e, err := htmlindex.Name(enc); e == "utf-8" || err != nil {
		return doc.Find("html"), nil
	}

	_, _ = f.Seek(0, 0)

	r := enc.NewDecoder().Reader(f)
	doc, err = goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	return doc.Find("html"), nil
}

func getEncoding(s *goquery.Selection) encoding.Encoding {
	var e encoding.Encoding
	s.Find("head").Find("meta").Each(func(i int, s *goquery.Selection) {
		if charset, ok := s.Attr("charset"); ok {
			enc, err := htmlindex.Get(charset)
			if err != nil {
				return
			}
			e = enc
		}
		if _, ok := s.Attr("http-equiv"); ok {
			if con, ok := s.Attr("content"); ok {
				charset := strings.TrimPrefix(con, "text/html; charset=")
				enc, err := htmlindex.Get(charset)
				if err != nil {
					return
				}
				e = enc
			}
		}
	})
	return e
}
