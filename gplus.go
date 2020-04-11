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
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type gpPage struct {
	*goquery.Selection
}

type gpComment struct {
	*goquery.Selection
}

func (p gpPage) canonicalUrl() string {
	var u string
	sel := p.Clone()
	sel.Find(".comments").Remove()
	sel.Find("span").Each(func(_ int, s *goquery.Selection) {
		if at, _ := s.Attr("itemprop"); at == "dateCreated" {
			u, _ = s.Parent().Attr("href")
		}
	})
	u = string(u[strings.LastIndex(u, "/")+1:])
	return u
}

func (p gpPage) content() pageContent {
	s := p.Find(".main-content").Clone()
	s.Find("br").ReplaceWithHtml("<p>")
	return pageContent{s}
}

func (p gpPage) date() time.Time {
	var d string
	sel := p.Clone()
	sel.Find(".comments").Remove()
	sel.Find("span").Each(func(_ int, s *goquery.Selection) {
		if at, _ := s.Attr("itemprop"); at == "dateCreated" {
			d = s.Text()
		}
	})
	dt, _ := time.Parse("2006-01-02T15:04:05-0700", d)
	return dt
}

func (p gpPage) title() string {
	return ""
}

func (p gpPage) tags() []string {
	return nil
}

func (p gpPage) webmentions() []byte {
	var mentions = struct {
		Type     string    `json:"type"`
		Name     string    `json:"name"`
		Children []mention `json:"children,omitempty"`
	}{Type: "feed", Name: "Webmentions"}

	mentions.Children = append(mentions.Children, p.reactions()...)

	p.Find(".comments").Find(".comment").Each(func(i int, s *goquery.Selection) {
		cmt := gpComment{s}
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

func (p gpPage) reactions() []mention {
	var mentions []mention
	mentions = append(mentions, p.getReactions(".resharers", "repost-of")...)
	mentions = append(mentions, p.getReactions(".plus-oners", "like-of")...)
	return mentions
}

func (p gpPage) getReactions(sel, typ string) []mention {
	var mentions []mention
	p.Find(sel).Children().Each(func(_ int, s *goquery.Selection) {
		u, _ := s.Attr("href")
		var a = author{"card", s.Text(), u, ""}
		var m = mention{
			Type:     "entry",
			Property: typ,
			Author:   a,
		}
		mentions = append(mentions, m)
	})
	return mentions
}

func (c gpComment) author() author {
	s := c.Find(".author")
	n := s.Text()
	u, _ := s.Attr("href")
	return author{"card", n, u, ""}
}

func (c gpComment) content() content {
	s := c.Find(".comment-content")
	t := s.Text()
	h, _ := s.Html()
	return content{t, h}
}

func (c gpComment) url() string {
	return ""
}

func (c gpComment) date() string {
	p := gpPage{c.Clone()}
	d := p.date()
	return d.String()

}
