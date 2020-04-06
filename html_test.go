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
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/htmlindex"
)

func TestHugo(t *testing.T) {
	tests := map[string]struct {
		file      string
		processor string
		want      string
	}{
		"diary_comments": {"diary_comments.htm", "diary", "diary_comments.md"},
		"diary_pic":      {"diary_pic.htm", "diary", "diary_pic.md"},
		"ljbackup":       {"ljbackup.html", "ljbackup", "ljbackup.md"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := loadHtmlFile(filepath.Join("testdata", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			var p page
			switch tc.processor {
			case "diary":
				p = diaryPage{s}
			case "ljbackup":
				p = ljbPage{s}
			default:
				t.Fatal("not implemented")
			}
			g := filepath.Join("testdata", tc.want)
			got := hugo(p, false)
			assertGolden(t, got, g)
		})
	}
}

func TestCanonicalUrl(t *testing.T) {
	tests := map[string]struct {
		file      string
		processor string
		want      string
	}{
		"diary_comments": {"diary_comments.htm", "diary", "p706232_soundcheck.htm"},
		"diary_pic":      {"diary_pic.htm", "diary", "p1225644_nachdenklichkeit.htm"},
		"ljbackup":       {"ljbackup.html", "ljbackup", "170041.html"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := loadHtmlFile(filepath.Join("testdata", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			var p page
			switch tc.processor {
			case "diary":
				p = diaryPage{s}
			case "ljbackup":
				p = ljbPage{s}
			default:
				t.Fatal("not implemented")
			}
			got := p.canonicalUrl()
			if tc.want != got {
				t.Fatalf("want %s, got %s\n", tc.want, got)
			}
		})
	}
}

func TestProcessImages(t *testing.T) {
	tests := map[string]struct {
		file      string
		processor string
		want      map[string]string
	}{
		"diary_comments": {"diary_comments.htm", "diary", map[string]string{}},
		"diary_pic":      {"diary_pic.htm", "diary", map[string]string{"image0": "https://secure.diary.ru/userdir/4/9/6/1/4961/95265.jpg"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := loadHtmlFile(filepath.Join("testdata", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			var p page
			switch tc.processor {
			case "diary":
				p = diaryPage{s}
			case "ljbackup":
				p = ljbPage{s}
			default:
				t.Fatal("not implemented")
			}
			cnt := p.content()
			got := cnt.processImages()

			if len(got) != len(tc.want) {
				t.Fatalf("want %s, got %s\n", tc.want, got)
			}

			for k, v := range tc.want {
				if v != got[k] {
					t.Fatalf("want %s, got %s\n", tc.want, got)
				}
			}
		})
	}
}

func TestWebmentions(t *testing.T) {
	tests := map[string]struct {
		file      string
		processor string
		want      string
	}{
		"diary_comments": {"diary_comments.htm", "diary", "diary_comments.json"},
		"diary_pic":      {"diary_pic.htm", "diary", "diary_pic.json"},
		"ljbackup":       {"ljbackup.html", "ljbackup", "ljbackup.json"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := loadHtmlFile(filepath.Join("testdata", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			var p page
			switch tc.processor {
			case "diary":
				p = diaryPage{s}
			case "ljbackup":
				p = ljbPage{s}
			default:
				t.Fatal("not implemented")
			}
			g := filepath.Join("testdata", tc.want)
			got := p.webmentions()
			assertGolden(t, got, g)
		})
	}
}

func TestGetEncoding(t *testing.T) {
	tests := map[string]struct {
		file string
		want string
	}{
		"known": {filepath.Join("testdata", "eter.html"), "utf-8"},
		"diary": {filepath.Join("testdata", "diary_comments.htm"), "windows-1251"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := loadHtml(t, tc.file)
			e := getEncoding(s)
			got, _ := htmlindex.Name(e)
			if tc.want != got {
				t.Fatalf("want %s, got %s\n", tc.want, got)
			}
		})
	}
}
