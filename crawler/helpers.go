package crawler

import (
	"math/rand"
	"strings"
	"sync"

	"github.com/Chocapikk/cewlai/words"
)

func defaultContextLimit(opts CrawlOptions) int {
	if opts.MaxContext > 0 {
		return opts.MaxContext
	}
	return 4000
}

func buildContextFromPages(pageContexts []string, limit int) string {
	if len(pageContexts) == 0 {
		return ""
	}

	shuffled := make([]string, len(pageContexts))
	copy(shuffled, pageContexts)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	perPage := limit / len(shuffled)

	var b strings.Builder
	for _, page := range shuffled {
		if b.Len() >= limit {
			break
		}
		chunk := page
		if len(chunk) > perPage {
			chunk = chunk[:perPage]
		}
		b.WriteString(chunk)
		b.WriteString(" ")
	}

	result := b.String()
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func parseByExtension(ext string, body []byte, wordSet map[string]struct{}, pageContexts *[]string) {
	switch ext {
	case ".txt", ".md", ".csv", ".log", ".conf", ".cfg", ".ini", ".yml", ".yaml":
		extractTextContent(body, wordSet, pageContexts)
		return
	case ".pdf":
		var mu sync.Mutex
		extractPDFMetadata(body, &mu, wordSet, false, "")
		return
	case ".docx", ".xlsx", ".pptx", ".dotx", ".potx", ".ppsx":
		var mu sync.Mutex
		extractOfficeMetadata(body, &mu, wordSet, false, "")
		return
	}

	for _, p := range parsers {
		for _, e := range p.exts {
			if ext == e {
				p.parse(body, wordSet)
				return
			}
		}
	}
}

func extractTextContent(body []byte, wordSet map[string]struct{}, pageContexts *[]string) {
	text := string(body)
	for _, w := range words.NormalizeAndSplit(text) {
		wordSet[w] = struct{}{}
	}
	if trimmed := strings.TrimSpace(text); trimmed != "" {
		*pageContexts = append(*pageContexts, trimmed)
	}
}

func mapKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
