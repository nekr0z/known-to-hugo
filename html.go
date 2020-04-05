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
	"encoding/json"
	"net/url"
	"os"
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
}

type diaryPage struct {
	*goquery.Selection
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

type diaryComment struct {
	*goquery.Selection
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
		//		"tags":           getTags(sel),
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

func (p diaryPage) canonicalUrl() string {
	u, _ := p.Find(".singlePost").Find(".urlLink").Find("a").Attr("href")
	uri, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(uri.Path, "/")
}

func (p diaryPage) title() string {
	return p.Find(".postTitle").Find("h1").Text()
}

func (p diaryPage) date() time.Time {
	s := p.Find(".singlePost")
	date := s.Find(".postDate").Text()
	d := strings.Split(date, ", ")
	if len(d) != 2 {
		return time.Now()
	}
	date = d[1]
	mes := map[string]string{
		"января":   "Jan",
		"февраля":  "Feb",
		"марта":    "Mar",
		"апреля":   "Apr",
		"мая":      "May",
		"июня":     "Jun",
		"июля":     "Jul",
		"августа":  "Aug",
		"сентября": "Sep",
		"октября":  "Oct",
		"ноября":   "Nov",
		"декабря":  "Dec",
	}
	for mr, me := range mes {
		date = strings.ReplaceAll(date, mr, me)
	}

	t := s.Find(".postTitle").Find("span").Text()

	dt, _ := time.ParseInLocation("2 Jan 2006 15:04", date+" "+t, time.Local)
	return dt
}

func (p diaryPage) content() pageContent {
	s := p.Find(".singlePost").Find(".postInner")
	return pageContent{s}
}

func (p diaryPage) webmentions() []byte {
	var mentions = struct {
		Type     string    `json:"type"`
		Name     string    `json:"name"`
		Children []mention `json:"children,omitempty"`
	}{Type: "feed", Name: "Webmentions"}

	p.Find(".singleComment").Each(func(i int, s *goquery.Selection) {
		cmt := diaryComment{s}
		m := getWebmention(cmt)
		mentions.Children = append(mentions.Children, m)
	})
	if len(mentions.Children) > 0 {
		b, err := json.MarshalIndent(mentions, "", " ")
		if err != nil {
			panic(err)
		}
		return b
	}
	return nil
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

func (dc diaryComment) author() author {
	n := dc.Find(".authorName").Text()
	p, _ := dc.Find(".commentAuthor").Find("img").Attr("src")
	return author{"card", n, "", p}
}

func (dc diaryComment) content() content {
	c := dc.Find(".postInner")
	t := strings.TrimSpace(c.Text())
	h, _ := c.Html()
	h = strings.TrimSpace(h)
	return content{t, h}
}

func (dc diaryComment) date() string {
	d := dc.Find(".postTitle").Find("span").Text()
	date, _ := time.ParseInLocation("2006-01-02 в 15:04", d, time.Local)
	return date.String()
}

func (dc diaryComment) url() string {
	return ""
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
