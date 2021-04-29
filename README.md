# known-to-hugo
a little tool to help migrate your [Known](https://withknown.com/) website to [Hugo](https://gohugo.io/)

[![Build Status](https://travis-ci.com/nekr0z/known-to-hugo.svg?branch=master)](https://travis-ci.com/nekr0z/known-to-hugo) [![codecov](https://codecov.io/gh/nekr0z/known-to-hugo/branch/master/graph/badge.svg)](https://codecov.io/gh/nekr0z/known-to-hugo) [![Go Report Card](https://goreportcard.com/badge/github.com/nekr0z/known-to-hugo)](https://goreportcard.com/report/github.com/nekr0z/known-to-hugo)

##### Table of Contents
* [Why](#why)
* [How to use](#how)
  * [Options](#command-line-options)
* [Development](#development)
* [Credits](#credits)

#### Help `known-to-hugo` get better!
Join the [development](#development) (or just [buy me a coffee](https://www.buymeacoffee.com/nekr0z), that helps, too).

## Why
[Known](https://withknown.com/) is great and has brought a lot of people to [IndieWeb](https://indieweb.org/). However, its export features are incomplete and have bugs. If you want to start using [Hugo](https://gohugo.io/) instead, you need to get all your content from the Known instance and save it so that Hugo can work with it (a simple MySQL dump wouldn't do). This tool here does exactly that.

## How
Change the theme on your Known website to the builtin "Solo" theme (it may work with other themes, but no one has tested it yet). While your Known site is still up and running, open your command line and run the tool. It will try to save all your posts neatly to Hugo-compatible markdown files. It will also generate a JSON file with webmentions for every page that has them, as well as try to download the images.

**Beware:** the files will be overwritten without asking!

### Command line options
```
-w [website]
```
the domain to look for, as in `evgenykuznetsov.org` or `some-user.withknown.com` or `https://my-awesome-known-site.info`.

```
-ww [path]
```
the part of website to try and download. Default is `/content/posts`, so only posts get downloaded. You can use `-ww ""` to download whatever content is listed on your home page, or `-ww /content/all` to try to get everything there is.

```
-p [directory]
```
the directory to save everything to. Default is `known_website` in your current directory.

```
-d
```
mark all the downloaded pages as drafts for Hugo.

```
-c [number]
```
number of pages to try to process simultaneously. Your server that runs Known might not like `known-to-hugo`'s attempt to download all the posts simultaneously (for example, the [DreamHost](https://www.dreamhost.com/) shared web hosting I use starts serving `503`s instead of pages when I try about 20 processes in parallel), so this option limits the number of pages processed in parallel. Default is `15`.

### Local backups processing
If you happen to have a local backup of your old blog, these are some experimental options for you:
```
-dir [path]
```
tells `known-to-hugo` to work with a local directory rather than a running website. The `-w` and `-ww` options are ignored in this case; site type (below) must be specified:
```
-type gplus
```
the local backup is Google Plus posts directory (as composed with Google Takeout), or
```
-type lj_backup
```
the local backup is a LiveJournal backup made with `ljArchive`, or
```
-type diary.ru
```
the local backup is a whole-site `wget` copy of a diary.ru-hosted blog.

## Development
Pull requests are always welcome!

## Credits
This software includes the following software or parts thereof:
* [goquery](https://github.com/PuerkitoBio/goquery) Copyright © 2012-2016, Martin Angers & Contributors
* [BurntSushi/toml](https://github.com/BurntSushi/toml) Copyright © 2013 TOML authors
* [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) Copyright © 2018 Johannes Kaufmann
* [cascadia](https://github.com/andybalholm/cascadia) Copyright © 2011 Andy Balholm. All rights reserved.
* [The Go Programming Language](https://golang.org) Copyright © 2009 The Go Authors
