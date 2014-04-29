package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	srcRoot string
)

func main() {
	fmt.Println("..chanzero..")

	// TODO Specfiy output folder
	flag.StringVar(&srcRoot, "src", "", "Path to root src file for site to build.")
	flag.Parse()

	if _, err := os.Stat(srcRoot); err != nil {
		fmt.Printf("Couldn't open file \"%v\" (%v)\n", srcRoot, err.Error())
		os.Exit(-1)
	}

	fmt.Println("Building site with root", srcRoot)

	site := importSite(srcRoot)
	exportSite(site)
}

type page struct {
	srcPath string
	html    []byte
}

func NewPage(path string) *page {
	return &page{path, nil}
}

// Return export filepath for given page.
// TODO Assumes md - not error checking!
func (page *page) destPath() string {
	basePath := strings.TrimSuffix(page.srcPath, filepath.Ext(page.srcPath))
	return basePath + ".html"
}

// Wrapped HtmlRenderer which gathers all links in markdown
// TODO Write post about wrapping/oeverriding functionality in Go.
type LinkGatheringHtmlRenderer struct {
	*blackfriday.Html

	linkedUrls []string
}

func NewLinkGatheringHtmlRenderer(renderer blackfriday.Renderer) *LinkGatheringHtmlRenderer {
	return &LinkGatheringHtmlRenderer{renderer.(*blackfriday.Html), make([]string, 0)}
}

func (html *LinkGatheringHtmlRenderer) AddLink(url string) {
	html.linkedUrls = append(html.linkedUrls, url)
}

func (html *LinkGatheringHtmlRenderer) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	html.AddLink(string(link))
	html.Html.AutoLink(out, link, kind)
}

func (html *LinkGatheringHtmlRenderer) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	html.AddLink(string(link))
	html.Html.Link(out, link, title, content)
}

// Given a single root page, load the page and follow all local src links,
// loading each page recursively, linked to the root.
func importSite(path string) *page {
	page := NewPage(path)

	mdsrc, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Couldn't load \"%v\": %v\n", path, err.Error())
	}

	// Set up a 'common' converter
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_LATEX_DASHES
	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	linkGatheringRenderer := NewLinkGatheringHtmlRenderer(renderer)

	extensions := 0
	//extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS
	extensions |= blackfriday.EXTENSION_HEADER_IDS

	page.html = blackfriday.Markdown(mdsrc, linkGatheringRenderer, extensions)

	fmt.Println("URLs:")
	for _, url := range linkGatheringRenderer.linkedUrls {
		fmt.Println(url)
	}

	return page
}

// Given a root page, export the entire site.
func exportSite(page *page) {
	ioutil.WriteFile(page.destPath(), page.html, 0x644)
}
