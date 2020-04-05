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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type diaryPage struct {
	*goquery.Selection
}

type diaryComment struct {
	*goquery.Selection
}

func diaryDir(input, output string) {
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
		p := diaryPage{s}
		url := p.canonicalUrl()
		if url == "" || url != filepath.Base(path) {
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
