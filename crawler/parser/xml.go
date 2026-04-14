package parser

import (
	"encoding/xml"
	"strings"

	"github.com/Chocapikk/cewlai/words"
)

func ExtractFromXML(body []byte, wordSet map[string]struct{}) {
	decoder := xml.NewDecoder(strings.NewReader(string(body)))
	decoder.Strict = false

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		cd, ok := tok.(xml.CharData)
		if !ok {
			continue
		}
		text := strings.TrimSpace(string(cd))
		if text == "" {
			continue
		}
		for _, w := range words.NormalizeAndSplit(text) {
			wordSet[w] = struct{}{}
		}
	}
}
