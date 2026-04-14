package parser

import (
	"strings"
	"sync"

	"github.com/Chocapikk/cewlai/words"
)

type ContentParser struct {
	Types []string
	Exts  []string
	Parse func(body []byte, wordSet map[string]struct{})
}

func MatchType(contentType, reqURL string, types []string, exts []string) bool {
	for _, t := range types {
		if strings.Contains(contentType, t) {
			return true
		}
	}
	lower := strings.ToLower(reqURL)
	for _, ext := range exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

var Parsers = []ContentParser{
	{[]string{"javascript", "ecmascript"}, []string{".js", ".mjs"}, func(body []byte, wordSet map[string]struct{}) {
		ExtractFromJS(body, wordSet)
	}},
	{[]string{"xml", "svg"}, []string{".xml", ".svg", ".rss", ".atom", ".sitemap"}, ExtractFromXML},
	{[]string{"json"}, []string{".json", ".webmanifest"}, ExtractFromJSON},
	{[]string{"css"}, []string{".css"}, ExtractFromCSS},
	{[]string{"text/vtt", "subrip"}, []string{".vtt", ".srt"}, ExtractSubtitles},
	{[]string{"audio", "video"}, []string{".mp3", ".mp4", ".ogg", ".flac", ".wav", ".m4a", ".webm"}, ExtractMediaMetadata},
}

func ParseByExtension(ext string, body []byte, wordSet map[string]struct{}, pageContexts *[]string) {
	switch ext {
	case ".txt", ".md", ".csv", ".log", ".conf", ".cfg", ".ini", ".yml", ".yaml":
		ExtractTextContent(body, wordSet, pageContexts)
		return
	case ".pdf":
		var mu sync.Mutex
		ExtractPDFMetadata(body, &mu, wordSet, false, "")
		return
	case ".docx", ".xlsx", ".pptx", ".dotx", ".potx", ".ppsx":
		var mu sync.Mutex
		ExtractOfficeMetadata(body, &mu, wordSet, false, "")
		return
	}

	for _, p := range Parsers {
		for _, e := range p.Exts {
			if ext == e {
				p.Parse(body, wordSet)
				return
			}
		}
	}
}

func ExtractTextContent(body []byte, wordSet map[string]struct{}, pageContexts *[]string) {
	text := string(body)
	for _, w := range words.NormalizeAndSplit(text) {
		wordSet[w] = struct{}{}
	}
	if trimmed := strings.TrimSpace(text); trimmed != "" {
		*pageContexts = append(*pageContexts, trimmed)
	}
}
