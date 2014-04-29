package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/gmacd/container/set"
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

	exportSite(srcRoot)
}

type page struct {
	srcPath    string
	html       []byte
	linkedUrls []string
}

func NewPage(path string) *page {
	return &page{path, nil, make([]string, 0)}
}

func (page *page) AddLink(url string) {
	page.linkedUrls = append(page.linkedUrls, url)
}

func replaceExtension(path, newExtention string) string {
	basePath := strings.TrimSuffix(path, filepath.Ext(path))
	return basePath + "." + newExtention
}

// Wrapped HtmlRenderer which gathers all links in markdown
// TODO Write post about wrapping/oeverriding functionality in Go.
type LinkGatheringHtmlRenderer struct {
	*blackfriday.Html

	page *page
}

func NewLinkGatheringHtmlRenderer(renderer blackfriday.Renderer, page *page) *LinkGatheringHtmlRenderer {
	return &LinkGatheringHtmlRenderer{renderer.(*blackfriday.Html), page}
}

func (html *LinkGatheringHtmlRenderer) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	html.page.AddLink(string(link))
	html.Html.AutoLink(out, link, kind)
}

func (html *LinkGatheringHtmlRenderer) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	html.page.AddLink(string(link))
	html.Html.Link(out, link, title, content)
}

// Given a single root page, load the page and follow all local src links,
// loading each page recursively, linked to the root.
func importPage(path string) *page {
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

	linkGatheringRenderer := NewLinkGatheringHtmlRenderer(renderer, page)

	extensions := 0
	//extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS
	extensions |= blackfriday.EXTENSION_HEADER_IDS

	page.html = blackfriday.Markdown(mdsrc, linkGatheringRenderer, extensions)

	return page
}

// Given a root page, export the entire site.
func exportSite(rootSrc string) {
	rootFile := filepath.Base(rootSrc)
	rootSrcPath := filepath.Dir(rootSrc)
	destSrcPath := filepath.Dir(rootSrcPath)

	exportedPages := set.NewSetOfValues()
	exportPage(rootFile, rootSrcPath, destSrcPath, exportedPages)
}

func exportPage(pageSrcPath, rootSrcPath, destSrcPath string, previouslyExportedPaths *set.Set) {
	if !previouslyExportedPaths.Contains(pageSrcPath) {
		previouslyExportedPaths.Add(pageSrcPath)

		fullSrcPath := rootSrcPath + "/" + pageSrcPath
		fullDestPath := destSrcPath + "/" + replaceExtension(pageSrcPath, "html")
		fmt.Println(" Exporting page  src:", fullSrcPath)
		fmt.Println("                dest:", fullDestPath)

		page := importPage(fullSrcPath)

		// Ensure destination exists
		os.MkdirAll(filepath.Dir(fullDestPath), 0755)

		ioutil.WriteFile(fullDestPath, page.html, 0644)

		// Export referenced pages
		for _, linkUrl := range page.linkedUrls {
			linkSrc := replaceExtension(linkUrl, "md")

			exportPage(linkSrc, rootSrcPath, destSrcPath, previouslyExportedPaths)
		}
	}
}
