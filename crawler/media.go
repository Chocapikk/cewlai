package crawler

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/Chocapikk/cewlai/words"
	"github.com/dhowden/tag"
)

var (
	vttTimestampRe = regexp.MustCompile(`^\d{2}:\d{2}[:\.][\d.]+\s*-->.*$`)
	srtTimestampRe = regexp.MustCompile(`^\d{2}:\d{2}:\d{2},\d{3}\s*-->.*$`)
	srtIndexRe     = regexp.MustCompile(`^\d+$`)
	vttTagRe       = regexp.MustCompile(`<[^>]+>`)
)

func extractMediaMetadata(body []byte, wordSet map[string]struct{}) {
	m, err := tag.ReadFrom(bytes.NewReader(body))
	if err != nil {
		return
	}

	for _, f := range []string{m.Title(), m.Artist(), m.Album(), m.AlbumArtist(), m.Genre(), m.Composer()} {
		if f == "" {
			continue
		}
		for _, w := range words.NormalizeAndSplit(f) {
			wordSet[w] = struct{}{}
		}
	}
}

func extractSubtitles(body []byte, wordSet map[string]struct{}) {
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "WEBVTT" {
			continue
		}
		if vttTimestampRe.MatchString(line) || srtTimestampRe.MatchString(line) || srtIndexRe.MatchString(line) {
			continue
		}
		if strings.HasPrefix(line, "NOTE") || strings.HasPrefix(line, "STYLE") {
			continue
		}
		line = vttTagRe.ReplaceAllString(line, "")
		for _, w := range words.NormalizeAndSplit(line) {
			wordSet[w] = struct{}{}
		}
	}
}
