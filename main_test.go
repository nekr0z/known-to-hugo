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
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

var (
	update = flag.Bool("update", false, "update .golden files")
)

func TestGetTitle(t *testing.T) {
	s := loadHtml(t, filepath.Join("testdata", "tired.html"))
	got := getTitle(s)
	want := "Двигаться дальше…"
	assertString(t, want, got)
}

func TestGetPermalink(t *testing.T) {
	s := loadHtml(t, filepath.Join("testdata", "tired.html"))
	got := getPermalink(s)
	want := "https://evgenykuznetsov.org/2020/двигаться-дальше"
	assertString(t, want, got)
}

func TestGetLikeOf(t *testing.T) {
	s := loadHtml(t, filepath.Join("testdata", "whatever.html"))
	got := getLikeOf(s)
	want := "https://habr.com/ru/post/491672/"
	assertString(t, want, got)
}

func TestGetDtPublished(t *testing.T) {
	s := loadHtml(t, filepath.Join("testdata", "tired.html"))
	got := getDtPublished(s)
	want := "2020-03-17T19:58:16+0000"
	assertString(t, want, got)
}

func TestGetWebmentions(t *testing.T) {
	s := loadHtml(t, filepath.Join("testdata", "eter.html"))
	g := filepath.Join("testdata", "eter.json")
	got, _ := getWebmentions(s)
	assertGolden(t, got, g)
}

func assertGolden(t *testing.T, actual []byte, golden string) {
	t.Helper()

	if *update {
		if _, err := os.Stat(golden); os.IsNotExist(err) {
			ioutil.WriteFile(golden, actual, 0644)
		} else {
			t.Log("file", golden, "exists, remove it to record new golden result")
		}
	}
	expected, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Error("no file:", golden)
	}

	if !bytes.Equal(actual, expected) {
		t.Fatalf("want:\n%s\ngot:\n%s\n", expected, actual)
	}
}

func assertString(t *testing.T, want, got string) {
	t.Helper()
	if got != want {
		t.Fatalf("want: %v\ngot: %v", want, got)
	}
}

func loadHtml(t *testing.T, path string) *goquery.Selection {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	return doc.Find("html")
}
