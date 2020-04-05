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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

		return nil
	})
}

func downloadImages(path string, images map[string]string) {
	for fn, url := range images {
		filename := filepath.Join(path, fn)
		if err := downloadFile(filename, url); err != nil {
			fmt.Printf("failed to fetch image: %s - %v", url, err)
		}
	}
}
