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
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ljbPage struct {
	*goquery.Selection
}

type ljbComment struct {
	*goquery.Selection
}

func (p ljbPage) canonicalUrl() string {
	u, _ := p.Find(".lesstop").Find("a").Attr("href")
	uri, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(uri.Path, "/")
}

func (p ljbPage) content() pageContent {
	s := p.Find("body").Find("p").Eq(1).Clone()
	s.Find("br").ReplaceWithHtml("<p>")
	t := s.Find("font").Eq(0)
	if t.Text() == p.title() {
		t.Remove()
	}
	t = s.Find("img")
	if a, _ := t.Attr("src"); a == "../../../img/icon_protected.gif" {
		t.Remove()
	}
	return pageContent{s}
}

func (p ljbPage) date() time.Time {
	d := p.Find("td").Eq(1).Find("font").Text()
	d = strings.TrimPrefix(d, "@ ")
	dt, _ := time.ParseInLocation("2006-01-02 15:04:05", d, time.Local)
	return dt
}

func (p ljbPage) title() string {
	t := p.Find("title").Text()
	ct := p.Find("body").Find("p").Eq(1).Find("font").Text()
	u := p.Find(".ljuser").Eq(0).Text()
	if t == u+": "+ct {
		return ct
	}
	return ""
}

func (p ljbPage) tags() []string {
	var t []string
	p.Find("td").Eq(3).Find("a").Each(func(_ int, s *goquery.Selection) {
		t = append(t, s.Text())
	})
	return t
}

func (p ljbPage) webmentions() []byte {
	var mentions = struct {
		Type     string    `json:"type"`
		Name     string    `json:"name"`
		Children []mention `json:"children,omitempty"`
	}{Type: "feed", Name: "Webmentions"}

	p.Find(".talk-comment").Each(func(i int, s *goquery.Selection) {
		cmt := ljbComment{s}
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

func (c ljbComment) author() author {
	s := c.Find(".ljuser").Find("a").Eq(1)
	n := s.Text()
	u, _ := s.Attr("href")
	return author{"card", n, u, ""}
}

func (c ljbComment) content() content {
	s := c.Find("td").Eq(-1).Clone()
	s.Find("font").Parent().Remove()
	t := s.Text()
	h, _ := s.Html()
	return content{t, h}
}

func (c ljbComment) url() string {
	u, _ := c.Find("td").Eq(1).Find("font").Eq(2).Find("a").Attr("href")
	return u
}

func (c ljbComment) date() string {
	d := c.Find("td").Eq(1).Find("font").Eq(1).Text()
	d = strings.TrimSuffix(d, " (local)")
	t, _ := time.ParseInLocation("2006-01-02 03:04 pm", d, time.Local)
	return t.String()

}
