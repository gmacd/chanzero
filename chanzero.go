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
	"regexp"
	"strings"
)

var (
	srcRoot string
	cssPath string

	pageSplitterRegex *regexp.Regexp
)

func main() {
	fmt.Printf("\n...chanzero...\n\n")

	// Split on at least 3 '/'
	pageSplitterRegex = regexp.MustCompile(`[/]{3,}`)

	// TODO Specfiy output folder
	flag.StringVar(&srcRoot, "src", "", "Path to root src file for site to build.")
	flag.Parse()

	if _, err := os.Stat(srcRoot); err != nil {
		fmt.Printf("Couldn't open file \"%v\" (%v)\n", srcRoot, err.Error())
		os.Exit(-1)
	}

	fmt.Printf("Building site with root: %v\n\n", srcRoot)

	exportSite(srcRoot)
}

type page struct {
	srcPath    string
	destPath   string
	html       []byte
	linkedUrls []string
	settings   map[string]string
}

func NewPage(srcPath, destPath string) *page {
	return &page{srcPath, destPath, nil, make([]string, 0), make(map[string]string)}
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
func (page *page) importPage() {
	fileContents, err := ioutil.ReadFile(page.srcPath)
	if err != nil {
		fmt.Printf("Couldn't load \"%v\": %v\n", page.srcPath, err.Error())
	}

	mdsrc := fileContents
	strs := pageSplitterRegex.Split(string(fileContents), -1)
	if len(strs) > 1 {
		parseSettings(strs[0], page.settings)
		handleGlobalSettings(page.settings)
		mdsrc = []byte(strs[1])
	}

	// Set up a 'common' converter
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_LATEX_DASHES
	htmlFlags |= blackfriday.HTML_COMPLETE_PAGE
	title := page.settings["Title"]
	renderer := blackfriday.HtmlRenderer(htmlFlags, title, cssPath)

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

		page := NewPage(
			rootSrcPath+"/"+pageSrcPath,
			destSrcPath+"/"+replaceExtension(pageSrcPath, "html"))

		fmt.Printf(" Exporting page  src: %v\n", page.srcPath)
		fmt.Printf("                dest: %v\n\n", page.destPath)

		page.importPage()

		// Ensure destination exists
		os.MkdirAll(filepath.Dir(page.destPath), 0755)

		ioutil.WriteFile(page.destPath, page.html, 0644)

		// Export local, valid markdown links
		for _, linkUrl := range page.linkedUrls {
			linkSrc := replaceExtension(linkUrl, "md")

			if canOpen(rootSrcPath + "/" + linkSrc) {
				exportPage(linkSrc, rootSrcPath, destSrcPath, previouslyExportedPaths)
			}
		}
	}
}

func parseSettings(str string, settings map[string]string) {
	for _, line := range strings.Split(str, "\n") {
		tokens := strings.SplitN(line, ":", 2)
		if len(tokens) == 2 {
			key := strings.TrimSpace(tokens[0])
			value := strings.TrimSpace(tokens[1])
			settings[key] = value
		}
	}
}

func handleGlobalSettings(pageSettings map[string]string) {
	if value, ok := pageSettings["SiteCss"]; ok {
		cssPath = value
	}
}

func canOpen(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
