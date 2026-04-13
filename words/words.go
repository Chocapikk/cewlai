package words

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	wordSplitter = regexp.MustCompile(`[^a-zA-Z0-9'\-]+`)
	hasNumbers   = regexp.MustCompile(`[0-9]`)
)

func NormalizeAndSplit(text string) []string {
	var out []string
	for _, w := range wordSplitter.Split(text, -1) {
		if w = strings.TrimSpace(w); w != "" {
			out = append(out, w)
		}
	}
	return out
}

func FilterWords(words []string, minLen, maxLen int, withNumbers bool) []string {
	var out []string
	for _, w := range words {
		if keepWord(w, minLen, maxLen, withNumbers) {
			out = append(out, w)
		}
	}
	return out
}

func keepWord(w string, minLen, maxLen int, withNumbers bool) bool {
	if len(w) < minLen {
		return false
	}
	if maxLen > 0 && len(w) > maxLen {
		return false
	}
	return withNumbers || !hasNumbers.MatchString(w)
}

func LowercaseWords(words []string) []string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = strings.ToLower(w)
	}
	return out
}

func DeduplicateWords(wordSets ...[]string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, set := range wordSets {
		collectUnique(set, seen, &result)
	}
	sort.Strings(result)
	return result
}

func collectUnique(words []string, seen map[string]struct{}, result *[]string) {
	for _, w := range words {
		if _, exists := seen[w]; exists {
			continue
		}
		seen[w] = struct{}{}
		*result = append(*result, w)
	}
}

func CountWords(words []string) map[string]int {
	counts := make(map[string]int)
	for _, w := range words {
		counts[w]++
	}
	return counts
}

func FormatWithCounts(words []string) []string {
	counts := CountWords(words)
	unique := DeduplicateWords(words)
	out := make([]string, len(unique))
	for i, w := range unique {
		out[i] = fmt.Sprintf("%d %s", counts[w], w)
	}
	return out
}

func GenerateGroups(words []string, groupSize int) []string {
	if groupSize < 2 || len(words) < groupSize {
		return nil
	}
	out := make([]string, 0, len(words)-groupSize+1)
	for i := 0; i <= len(words)-groupSize; i++ {
		out = append(out, strings.Join(words[i:i+groupSize], " "))
	}
	return out
}
