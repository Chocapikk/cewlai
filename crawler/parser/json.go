package parser

import (
	"encoding/json"

	"github.com/Chocapikk/cewlai/words"
)

func ExtractFromJSON(body []byte, wordSet map[string]struct{}) {
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return
	}
	walkJSON(raw, wordSet)
}

func walkJSON(v any, wordSet map[string]struct{}) {
	switch val := v.(type) {
	case string:
		for _, w := range words.NormalizeAndSplit(val) {
			wordSet[w] = struct{}{}
		}
	case map[string]any:
		for k, child := range val {
			for _, w := range words.NormalizeAndSplit(k) {
				wordSet[w] = struct{}{}
			}
			walkJSON(child, wordSet)
		}
	case []any:
		for _, child := range val {
			walkJSON(child, wordSet)
		}
	}
}
