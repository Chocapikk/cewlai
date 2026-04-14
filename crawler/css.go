package crawler

import (
	"regexp"
	"strings"

	"github.com/Chocapikk/cewlai/words"
)

var (
	cssSelectorRe = regexp.MustCompile(`[.#][a-zA-Z][a-zA-Z0-9_-]*`)
	cssVarRe      = regexp.MustCompile(`--[a-zA-Z][a-zA-Z0-9_-]*`)
	cssURLRe      = regexp.MustCompile(`url\(['"]?([^'")\s]+)['"]?\)`)
	cssCommentRe  = regexp.MustCompile(`/\*([^*]|\*[^/])*\*/`)
)

func extractFromCSS(body []byte, wordSet map[string]struct{}) {
	content := string(body)

	for _, comment := range cssCommentRe.FindAllString(content, -1) {
		text := strings.Trim(comment, "/* ")
		for _, w := range words.NormalizeAndSplit(text) {
			wordSet[w] = struct{}{}
		}
	}

	for _, match := range cssSelectorRe.FindAllString(content, -1) {
		name := strings.TrimLeft(match, ".#")
		for _, w := range words.NormalizeAndSplit(name) {
			wordSet[w] = struct{}{}
		}
	}

	for _, match := range cssVarRe.FindAllString(content, -1) {
		name := strings.TrimLeft(match, "-")
		for _, w := range words.NormalizeAndSplit(name) {
			wordSet[w] = struct{}{}
		}
	}

	for _, match := range cssURLRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			for _, w := range words.NormalizeAndSplit(match[1]) {
				wordSet[w] = struct{}{}
			}
		}
	}
}
