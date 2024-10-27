package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	_ "io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

type Config struct {
	inputFile     string
	outputFile    string
	removeJS      bool
	baseDir       string
	processedURLs map[string]bool
}

func main() {
	// Parse command line flags
	inputFile := flag.String("input", "", "Path to input HTML file (required)")
	outputFile := flag.String("output", "", "Path to output HTML file (required)")
	removeJS := flag.Bool("remove-js", false, "Remove all JavaScript code and references")
	flag.Parse()

	if *inputFile == "" || *outputFile == "" {
		log.Fatal("Both input and output file paths are required")
	}

	// Create configuration
	config := &Config{
		inputFile:     *inputFile,
		outputFile:    *outputFile,
		removeJS:      *removeJS,
		baseDir:       filepath.Dir(*inputFile),
		processedURLs: make(map[string]bool),
	}

	// Process the HTML file
	if err := processHTML(config); err != nil {
		log.Fatal(err)
	}

	absPath, err := filepath.Abs(*outputFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Processed HTML file written to: %s\n", absPath)
}

func processHTML(config *Config) error {
	// Read input file
	file, err := os.Open(config.inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer file.Close()

	// Parse HTML
	doc, err := html.Parse(file)
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}

	// Process the document
	processNode(doc, config)

	// Create output file
	outFile, err := os.Create(config.outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outFile.Close()

	// Write the processed HTML
	if err := html.Render(outFile, doc); err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}

	return nil
}

func processNode(n *html.Node, config *Config) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "script":
			if config.removeJS {
				// Mark node for removal
				n.Parent.RemoveChild(n)
				return
			}
		case "link":
			if isPreloadJS(n) && config.removeJS {
				// Remove preload links for JS files
				n.Parent.RemoveChild(n)
				return
			} else if isStylesheet(n) {
				// Embed CSS
				embedCSS(n, config)
			}
		}

		// Remove inline JavaScript attributes if removeJS is true
		if config.removeJS {
			removeInlineJS(n)
		}
	}

	// Process child nodes
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		processNode(c, config)
		c = next
	}
}

func isPreloadJS(n *html.Node) bool {
	var rel, as string
	for _, a := range n.Attr {
		switch a.Key {
		case "rel":
			rel = a.Val
		case "as":
			as = a.Val
		}
	}
	return rel == "preload" && as == "script"
}

func isStylesheet(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "rel" && a.Val == "stylesheet" {
			return true
		}
	}
	return false
}

func embedCSS(n *html.Node, config *Config) {
	var href string
	for _, a := range n.Attr {
		if a.Key == "href" {
			href = a.Val
			break
		}
	}

	if href == "" {
		return
	}

	// Handle paths starting with /_next
	if strings.HasPrefix(href, "/_next") {
		href = filepath.Join(config.baseDir, href)
	}

	// Read CSS file
	cssContent, err := os.ReadFile(href)
	if err != nil {
		log.Printf("Warning: Could not read CSS file %s: %v", href, err)
		return
	}

	// Create new style node
	styleNode := &html.Node{
		Type: html.ElementNode,
		Data: "style",
		Attr: []html.Attribute{
			{Key: "type", Val: "text/css"},
		},
	}

	// Add CSS content
	styleNode.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: string(cssContent),
	})

	// Replace link node with style node
	n.Parent.InsertBefore(styleNode, n)
	n.Parent.RemoveChild(n)
}

func removeInlineJS(n *html.Node) {
	// List of JavaScript event attributes to remove
	jsAttributes := []string{
		"onclick", "onload", "onunload", "onchange", "onsubmit", "onreset",
		"onselect", "onblur", "onfocus", "onkeydown", "onkeypress", "onkeyup",
		"onmouseover", "onmouseout", "onmousedown", "onmouseup", "onmousemove",
	}

	// Create new attribute list without JavaScript events
	newAttrs := make([]html.Attribute, 0, len(n.Attr))
	for _, attr := range n.Attr {
		isJSAttr := false
		for _, jsAttr := range jsAttributes {
			if attr.Key == jsAttr {
				isJSAttr = true
				break
			}
		}
		if !isJSAttr {
			newAttrs = append(newAttrs, attr)
		}
	}
	n.Attr = newAttrs
}

func embedImage(n *html.Node, config *Config) {
	var src string
	for _, a := range n.Attr {
		if a.Key == "src" {
			src = a.Val
			break
		}
	}

	if src == "" || config.processedURLs[src] {
		return
	}

	// Handle paths starting with /_next
	if strings.HasPrefix(src, "/_next") {
		src = filepath.Join(config.baseDir, src)
	}

	// Read image file
	imgContent, err := os.ReadFile(src)
	if err != nil {
		log.Printf("Warning: Could not read image file %s: %v", src, err)
		return
	}

	// Determine MIME type
	mimeType := mime.TypeByExtension(filepath.Ext(src))
	if mimeType == "" {
		mimeType = "image/png" // Default to PNG if can't determine
	}

	// Convert to base64
	b64Content := base64.StdEncoding.EncodeToString(imgContent)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64Content)

	// Update src attribute
	for i, a := range n.Attr {
		if a.Key == "src" {
			n.Attr[i].Val = dataURL
			break
		}
	}

	config.processedURLs[src] = true
}
