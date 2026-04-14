package crawler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Chocapikk/cewlai/crawler/parser"
)

type discoveredFile struct {
	path string
	name string
}

func processFiles(proto string, files []discoveredFile, wordSet map[string]struct{}, opts CrawlOptions, download func(discoveredFile) ([]byte, error)) ([]string, int) {
	var mu sync.Mutex
	var pageContexts []string
	var processed atomic.Int32
	total := len(files)

	for _, f := range files {
		body, err := download(f)
		if err != nil || len(body) == 0 {
			continue
		}

		ext := strings.ToLower(filepath.Ext(f.name))
		mu.Lock()
		parser.ParseByExtension(ext, body, wordSet, &pageContexts)
		mu.Unlock()

		n := processed.Add(1)
		fmt.Fprintf(os.Stderr, "\r[*] %s: %d/%d files processed", proto, n, total)
	}

	fmt.Fprintf(os.Stderr, "\r\033[K")
	return pageContexts, int(processed.Load())
}

func buildFileResult(proto, addr string, wordSet map[string]struct{}, pageContexts []string, filesProcessed int, opts CrawlOptions) *CrawlResult {
	return &CrawlResult{
		Words:   mapKeys(wordSet),
		Context: buildContextFromPages(pageContexts, defaultContextLimit(opts)),
		URL:     proto + "://" + addr,
		Pages:   filesProcessed,
	}
}
