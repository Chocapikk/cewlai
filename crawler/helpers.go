package crawler

import (
	"math/rand"
	"strings"
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

func mapKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
