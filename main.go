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

//go:generate go run version_generate.go

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

var (
	outputDir, website, what string
	concurrency              int
	draft                    bool
)

const frontMatterSeparator = "+++\n"

type content struct {
	Text string `json:"text,omitempty"`
	Html string `json:"html,omitempty"`
}
type author struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"name,omitempty"`
	Url   string `json:"url,omitempty"`
	Photo string `json:"photo,omitempty"`
}
type mention struct {
	Type     string  `json:"type,omitempty"`
	Property string  `json:"wm-property,omitempty"`
	Author   author  `json:"author"`
	Url      string  `json:"url"`
	Date     string  `json:"wm-received"`
	Content  content `json:"content,omitempty"`
}

func main() {
	fmt.Printf("known-to-hugo version %s\n", version)
	flag.BoolVar(&draft, "d", false, "mark each entry as draft")
	flag.IntVar(&concurrency, "c", 15, "number of pages to process simultaneously")
	flag.StringVar(&website, "w", "example.site", "website to scrape")
	flag.StringVar(&outputDir, "p", "./known_website", "directory to save the results to")
	flag.StringVar(&what, "ww", "/content/posts", "section of the site to scrape, use \"\" for default content)")
	flag.Parse()
	if !strings.HasPrefix(website, "http://") && !strings.HasPrefix(website, "https://") {
		website = "http://" + website
	}

	pages := getPostLinks(website + what)
	defImg := getDefaultImage(website)
	processPages(pages, defImg)
	fmt.Println("all done!")
}

func processPages(pages []string, defaultImage string) {
	var errors []error
	errC := make(chan error)
	counter := 0
	sem := make(chan struct{}, concurrency)
	for _, page := range pages {
		counter = counter + 1
		go processPage(sem, errC, page, defaultImage)
	}
	for i := 0; i < counter; i++ {
		err := <-errC
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		fmt.Printf("the following %v errors occured:\n", len(errors))
		for _, err := range errors {
			fmt.Println(err)
		}
	}
}

func processPage(sem chan struct{}, errC chan error, url, defaultImage string) {
	sem <- struct{}{}
	defer func() { <-sem }()

	fmt.Printf("processing %s\n", url)
	d, err := getPage(url)
	if err != nil {
		err := fmt.Errorf("could not process %s - %w", url, err)
		errC <- err
		return
	}
	sel := d.Find("html")
	year := getPostYear(sel)
	slug := getPostSlug(url, year)
	dir := filepath.Join(outputDir, year, slug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	processWebmentions(sel, dir)
	processImages(sel, dir)
	processLinksToFiles(sel, dir)
	processLinksToOwnSite(sel)
	b := parsePage(sel, defaultImage)
	fn := filepath.Join(dir, "index.md")
	if err := ioutil.WriteFile(fn, b, 0644); err != nil {
		panic(err)
	}
	errC <- nil
}

func processWebmentions(sel *goquery.Selection, path string) {
	if b, ok := getWebmentions(sel); ok {
		fn := filepath.Join(path, "webmentions.json")
		if err := ioutil.WriteFile(fn, b, 0644); err != nil {
			panic(err)
		}
	}
}

func getWebmentions(sel *goquery.Selection) ([]byte, bool) {
	var mentions = struct {
		Type     string    `json:"type"`
		Name     string    `json:"name"`
		Children []mention `json:"children,omitempty"`
	}{Type: "feed", Name: "Webmentions"}

	sel.Find(".annotations").Find(".idno-annotation").Each(func(i int, s *goquery.Selection) {
		m := getMention(s)
		mentions.Children = append(mentions.Children, m)
	})
	if len(mentions.Children) > 0 {
		b, err := json.MarshalIndent(mentions, "", " ")
		if err != nil {
			panic(err)
		}
		return b, true
	} else {
		return nil, false
	}
}

func getMention(sel *goquery.Selection) mention {
	var m = mention{
		Type:   "entry",
		Author: getMentionAuthor(sel),
	}
	if c, ok := getMentionContent(sel); ok {
		m.Content = c
	}
	if t, ok := getMentionType(sel); ok {
		m.Property = t
	}
	m.Url, m.Date = getMentionSource(sel)
	return m
}

func getMentionType(sel *goquery.Selection) (string, bool) {
	s := sel.Find(".idno-annotation-content").Find("p").Eq(0)
	if s.Parent().Is(".e-content") {
		return "comment", false
	}
	a := s.Find("a").Eq(1).Text()
	if strings.HasPrefix(a, "reshared") {
		return "repost-of", true
	}
	a = s.Text()
	if strings.HasSuffix(strings.TrimSpace(a), "liked this post") {
		return "like-of", true
	}
	return a, true
}

func getMentionSource(sel *goquery.Selection) (url, date string) {
	s := sel.Find(".idno-annotation-content").Find("a").Eq(-2)
	d := s.Text()
	dt, err := time.Parse("Jan 02 2006", d)
	if err != nil {
		date = d
	} else {
		date = dt.Format("2006-01-02")
	}
	url, _ = s.Attr("href")
	return
}

func getMentionAuthor(sel *goquery.Selection) author {
	p, _ := sel.Find(".idno-annotation-image").Find("img").Attr("src")
	au := sel.Find(".idno-annotation-content").Find("a").Eq(0)
	n := au.Text()
	u, _ := au.Attr("href")
	s := sel.Find(".h-card")
	if s.Is(".h-card") {
		n = s.Find(".p-name").Text()
		u, _ = s.Find(".p-name").Attr("href")
		p, _ = s.Find(".u-photo").Attr("href")
	}
	return author{"card", n, u, p}
}

func getMentionContent(sel *goquery.Selection) (content, bool) {
	cont := sel.Find(".e-content")
	if cont.Is(".e-content") {
		text := cont.Text()
		html, _ := cont.Html()
		return content{text, html}, true
	}
	return content{}, false
}

func processLinksToOwnSite(sel *goquery.Selection) {
	prefix := strings.TrimSuffix(website, "/")
	sel.Find(".e-content").Find("a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		rel := strings.TrimPrefix(link, prefix)
		s.SetAttr("href", rel)
	})
}

func processLinksToFiles(sel *goquery.Selection, dir string) {
	fPrefix := strings.TrimSuffix(website, "/") + "/file/"
	se := sel.Find(".e-content")
	se.Find("a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		if strings.HasPrefix(link, fPrefix) {
			pts := strings.Split(link, "/")
			fn := strconv.Itoa(i) + pts[len(pts)-1]
			filename := filepath.Join(dir, fn)
			if err := downloadFile(filename, link); err != nil {
				fmt.Printf("failed to fetch asset: %s - %v", link, err)
				return
			}
			changeHrefs(se, link, fn)
		}
	})
}

func processImages(sel *goquery.Selection, dir string) {
	featured := getFeaturedImage(sel)
	se := sel.Find(".e-content")
	se.Find("img").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("src")
		photoUrl := strings.TrimSuffix(link, "/thumb.jpg")
		fn := "image" + strconv.Itoa(i)
		filename := filepath.Join(dir, fn)
		if err := downloadFile(filename, photoUrl); err != nil {
			fmt.Printf("failed to fetch image: %s - %v", photoUrl, err)
			return
		}

		// fix hrefs
		changeHrefs(se, link, fn)

		// fix featured image, too
		if link == featured {
			sel.Find("meta").Each(func(i int, s *goquery.Selection) {
				v, _ := s.Attr("property")
				if v == "og:image" {
					s.SetAttr("content", fn)
				}
			})
		}
		s.SetAttr("src", fn)
	})
}

func changeHrefs(se *goquery.Selection, link, fn string) {
	se.Find("a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if href == link {
			s.SetAttr("href", fn)
		}
	})
}

func getPostSlug(uri, year string) string {
	u, err := url.Parse(uri)
	if err != nil {
		// Known is buggy as hell
		uf := fixURL(uri)
		u, err = url.Parse(uf)
		if err != nil {
			panic(err)
		}
	}
	slug := strings.TrimPrefix(u.Path, "/"+year+"/")
	return slug
}

func getPostYear(sel *goquery.Selection) string {
	dateString := getDtPublished(sel)
	date, err := time.Parse("2006-01-02T15:04:05-0700", dateString)
	if err != nil {
		panic(err)
	}
	return date.Format("2006")
}

func getPostLinks(url string) []string {
	var links []string
	next := true
	for next {
		fmt.Printf("Processing %s\n", url)
		d, err := getPage(url)
		if err != nil {
			break
		}
		s := d.Find("html")
		var nurl string
		nurl, next = s.Find(".older").Find("a").Attr("href")
		// Known is buggy as hell:
		if nurl == url {
			next = false
		}
		url = nurl
		links = append(links, getPostLinksFromPage(s)...)
	}
	return links
}

func getPostLinksFromPage(sel *goquery.Selection) []string {
	var links []string
	sel.Find(".idno-entry").Each(func(i int, s *goquery.Selection) {
		link := getPermalink(s)
		links = append(links, link)
	})
	return links
}

func parsePage(sel *goquery.Selection, defaultImage string) []byte {
	var b []byte
	b = append(b, getFrontMatter(sel, defaultImage)...)
	b = append(b, []byte(getMd(sel))...)
	return b
}

func getFrontMatter(sel *goquery.Selection, defaultImage string) []byte {
	featured := getFeaturedImage(sel)
	if featured == defaultImage {
		featured = ""
	}

	var frontMatter = map[string]interface{}{
		"title":          getTitle(sel),
		"aliases":        []string{getRelPermalink(sel)},
		"date":           getDtPublished(sel),
		"featured_image": featured,
		"tags":           getTags(sel),
		"reply_to":       getInReply(sel),
		"posse":          getSyndications(sel),
		"like_of":        getLikeOf(sel),
		"draft":          draft,
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

func getMd(sel *goquery.Selection) string {
	c := sel.Clone()
	converter := md.NewConverter("", true, nil)
	c.Find(".annotations").Remove()
	c.Find(".p-category").Remove()
	got := converter.Convert(c.Find(".e-content"))
	return got
}

func getInReply(sel *goquery.Selection) []string {
	var irt []string
	sel.Find(".u-in-reply-to").Each(func(i int, s *goquery.Selection) {
		rep, _ := s.Attr("href")
		irt = append(irt, rep)
	})
	return irt
}

func getLikeOf(sel *goquery.Selection) string {
	s := sel.Find(".u-like-of")
	// Known is awesome :/
	var like string
	if l, ok := s.Attr("href"); ok {
		if l != "" {
			like = l
		} else {
			like, _ = sel.Find(".unfurl").Attr("data-url")
		}
	}
	return like
}

func getSyndications(sel *goquery.Selection) []string {
	var irt []string
	sel.Find(".u-syndication").Each(func(i int, s *goquery.Selection) {
		rep, _ := s.Attr("href")
		irt = append(irt, rep)
	})
	return irt
}

func getTags(sel *goquery.Selection) []string {
	var tags []string
	sel.Find(".p-category").Each(func(i int, s *goquery.Selection) {
		tag := strings.TrimPrefix(s.Text(), "#")
		tags = append(tags, tag)
	})
	return tags
}

func getFeaturedImage(sel *goquery.Selection) string {
	var img string
	sel.Find("meta").Each(func(i int, s *goquery.Selection) {
		v, _ := s.Attr("property")
		if v == "og:image" {
			img, _ = s.Attr("content")
		}
	})
	return img
}

func getDefaultImage(url string) string {
	d, err := getPage(url)
	if err != nil {
		return ""
	}
	s := d.Find("html")
	return getFeaturedImage(s)
}

func getTitle(sel *goquery.Selection) string {
	v := sel.Find(".idno-body").Find(".p-name").Find("a").Text()
	return v
}

func getPermalink(sel *goquery.Selection) string {
	v, _ := sel.Find(".permalink").Find(".u-url").Attr("href")
	v, _ = url.PathUnescape(v)
	return v
}

func getRelPermalink(sel *goquery.Selection) string {
	link := getPermalink(sel)
	u, err := url.Parse(link)
	if err != nil {
		// Known is buggy as hell
		uf := fixURL(link)
		u, err = url.Parse(uf)
		if err != nil {
			panic(err)
		}
	}
	return u.Path
}

func getDtPublished(sel *goquery.Selection) string {
	v, _ := sel.Find(".dt-published").Attr("datetime")
	return v
}

func getPage(uri string) (*goquery.Document, error) {
	res, err := http.Get(uri)
	if err != nil {
		// Known is buggy as hell
		u := fixURL(uri)
		res, err = http.Get(u)
		if err != nil {
			return nil, fmt.Errorf("can not parse URL")
		}
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err := fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
		return nil, fmt.Errorf("can not get page: %w", err)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		panic(err)
	}

	return doc, nil
}

func fixURL(uri string) string {
	pth := strings.TrimPrefix(uri, website)
	pth = strings.TrimPrefix(pth, "/")
	comps := strings.SplitN(pth, "/", 2)
	if len(comps) < 2 {
		return ""
	}
	parts := []string{website, comps[0], url.PathEscape(comps[1])}
	res := strings.Join(parts, "/")
	return res
}

func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
