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
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ljbPage struct {
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
