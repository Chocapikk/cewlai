package parser

import (
	"fmt"

	"github.com/BishopFox/jsluice"
	"github.com/Chocapikk/cewlai/words"
)

func ExtractFromJS(body []byte, wordSet map[string]struct{}) []string {
	analyzer := jsluice.NewAnalyzer(body)
	analyzer.AddSecretMatchers(jsluice.AllSecretMatchers())

	var urls []string
	for _, url := range analyzer.GetURLs() {
		for _, w := range words.NormalizeAndSplit(url.URL) {
			wordSet[w] = struct{}{}
		}
		urls = append(urls, url.URL)
	}

	for _, secret := range analyzer.GetSecrets() {
		for _, w := range words.NormalizeAndSplit(fmt.Sprint(secret.Data)) {
			wordSet[w] = struct{}{}
		}
	}

	return urls
}
