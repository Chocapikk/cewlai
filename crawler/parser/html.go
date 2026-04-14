package parser

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var WordAttrs = []string{
	"alt", "title", "placeholder", "aria-label", "aria-description",
	"data-title", "data-name", "data-label", "data-value",
	"content", "value", "label", "summary",
}

func ExtractTitle(doc *goquery.Document) string {
	if t := strings.TrimSpace(doc.Find("title").First().Text()); t != "" {
		return t
	}
	return ""
}

func ExtractAttrs(doc *goquery.Document, addWords func(string), addContext func(string)) {
	doc.Find("meta[name=description], meta[name=keywords]").Each(func(_ int, sel *goquery.Selection) {
		if content, exists := sel.Attr("content"); exists && content != "" {
			addWords(content)
			addContext(content)
		}
	})

	doc.Find("*").Each(func(_ int, sel *goquery.Selection) {
		for _, attr := range WordAttrs {
			if val, exists := sel.Attr(attr); exists && val != "" {
				addWords(val)
			}
		}
	})
}

func ExtractComments(doc *goquery.Document, addWords func(string)) {
	doc.Contents().Each(func(_ int, sel *goquery.Selection) {
		for _, node := range sel.Nodes {
			extractCommentsFromNode(node, addWords)
		}
	})
}

func extractCommentsFromNode(node *html.Node, addWords func(string)) {
	if node.Type == html.CommentNode {
		addWords(strings.TrimSpace(node.Data))
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		extractCommentsFromNode(child, addWords)
	}
}

func ExtractBodyText(doc *goquery.Document, addWords func(string), addContext func(string)) {
	doc.Find("*").Each(func(_ int, sel *goquery.Selection) {
		sel.PrependHtml(" ")
	})
	bodyText := doc.Text()
	addWords(bodyText)
	addContext(bodyText)
}

func ExtractEmails(doc *goquery.Document, addEmail func(string)) {
	doc.Find("a[href^='mailto:']").Each(func(_ int, sel *goquery.Selection) {
		if href, exists := sel.Attr("href"); exists {
			email := strings.TrimPrefix(href, "mailto:")
			email = strings.Split(email, "?")[0]
			addEmail(email)
		}
	})

	bodyText := doc.Text()
	for _, e := range ExtractEmailsFromText(bodyText) {
		addEmail(e)
	}
}

func FollowLinks(doc *goquery.Document, visit func(string)) {
	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		if val, exists := sel.Attr("href"); exists {
			visit(val)
		}
	})
}

type Resource struct {
	Query string
	Attr  string
}

var Resources = []Resource{
	{"script[src]", "src"},
	{"link[href]", "href"},
	{"img[src]", "src"},
	{"iframe[src]", "src"},
	{"source[src]", "src"},
	{"video[src]", "src"},
	{"audio[src]", "src"},
	{"track[src]", "src"},
}

func FollowResources(doc *goquery.Document, visit func(string)) {
	for _, res := range Resources {
		doc.Find(res.Query).Each(func(_ int, sel *goquery.Selection) {
			if val, exists := sel.Attr(res.Attr); exists {
				visit(val)
			}
		})
	}
}
