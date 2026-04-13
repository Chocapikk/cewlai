package crawler

import (
	"regexp"
	"strings"
)

var (
	emailRe = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}`)

	obfuscatedRe = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+\s*[\[\(<{]?\s*(?:at|@)\s*[\]\)>}]?\s*[a-z0-9.\-]+\s*[\[\(<{]?\s*(?:dot|\.)\s*[\]\)>}]?\s*[a-z]{2,4}`)

	atReplacer  = regexp.MustCompile(`(?i)\s*[\[\(<{]?\s*\bat\b\s*[\]\)>}]?\s*`)
	dotReplacer = regexp.MustCompile(`(?i)\s*[\[\(<{]?\s*\bdot\b\s*[\]\)>}]?\s*`)
)

func extractEmailsFromText(text string) []string {
	seen := make(map[string]struct{})
	var out []string

	collect := func(email string) {
		if _, exists := seen[email]; exists {
			return
		}
		seen[email] = struct{}{}
		out = append(out, email)
	}

	for _, m := range emailRe.FindAllString(text, -1) {
		collect(strings.ToLower(m))
	}

	for _, m := range obfuscatedRe.FindAllString(text, -1) {
		if email := deobfuscateEmail(m); email != "" {
			collect(email)
		}
	}

	return out
}

func deobfuscateEmail(raw string) string {
	result := atReplacer.ReplaceAllString(raw, "@")
	result = dotReplacer.ReplaceAllString(result, ".")
	result = strings.ToLower(strings.TrimSpace(result))
	if emailRe.MatchString(result) {
		return result
	}
	return ""
}
