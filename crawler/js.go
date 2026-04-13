package crawler

import (
	"fmt"

	"github.com/BishopFox/jsluice"
	"github.com/Chocapikk/cewlai/words"
)

func extractFromJS(body []byte, wordSet map[string]struct{}) {
	analyzer := jsluice.NewAnalyzer(body)
	analyzer.AddSecretMatchers(jsluice.AllSecretMatchers())

	for _, url := range analyzer.GetURLs() {
		for _, w := range words.NormalizeAndSplit(url.URL) {
			wordSet[w] = struct{}{}
		}
	}

	for _, secret := range analyzer.GetSecrets() {
		for _, w := range words.NormalizeAndSplit(fmt.Sprint(secret.Data)) {
			wordSet[w] = struct{}{}
		}
	}
}
