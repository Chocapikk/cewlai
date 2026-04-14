package crawler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Chocapikk/cewlai/crawler/parser"
	"github.com/Chocapikk/cewlai/words"
)

type discoveredFile struct {
	path string
	name string
}

func processFiles(proto string, files []discoveredFile, wordSet map[string]struct{}, opts CrawlOptions, download func(discoveredFile) ([]byte, error)) ([]string, []string, int) {
	var mu sync.Mutex
	var pageContexts []string
	var secrets []string
	var processed atomic.Int32

	var scanner *parser.SecretScanner
	if opts.ExtractSecrets {
		scanner = parser.NewSecretScanner()
	}

	if opts.MaxFiles > 0 && len(files) > opts.MaxFiles {
		files = files[:opts.MaxFiles]
	}
	total := len(files)

	for _, f := range files {
		body, err := download(f)
		if err != nil || len(body) == 0 {
			continue
		}

		if opts.DumpDir != "" {
			dumpFile(opts.DumpDir, f.path, body)
		}

		ext := strings.ToLower(filepath.Ext(f.name))
		mu.Lock()
		parser.ParseByExtension(ext, body, wordSet, &pageContexts)
		mu.Unlock()

		if scanner != nil {
			findings := scanner.Scan(body, f.path)
			mu.Lock()
			for _, s := range findings {
				secrets = append(secrets, fmt.Sprintf("[%s] %s (source: %s)", s.DetectorName, s.Raw, s.Source))
			}
			mu.Unlock()
		}

		n := processed.Add(1)
		fmt.Fprintf(os.Stderr, "\r[*] %s: %d/%d files processed", proto, n, total)
	}

	fmt.Fprintf(os.Stderr, "\r\033[K")
	return pageContexts, secrets, int(processed.Load())
}

func dumpFile(dir, filePath string, data []byte) {
	cleaned := filepath.FromSlash(filePath)
	dest := filepath.Join(dir, cleaned)
	abs, err := filepath.Abs(dest)
	if err != nil {
		return
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	if !strings.HasPrefix(abs, absDir+string(filepath.Separator)) {
		return
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(abs, data, 0o644)
}

func addNamesToWordSet(name string, wordSet map[string]struct{}) {
	for _, w := range words.NormalizeAndSplit(name) {
		wordSet[w] = struct{}{}
	}
}

func buildFileResult(proto, addr string, wordSet map[string]struct{}, pageContexts []string, secrets []string, filesProcessed int, opts CrawlOptions) *CrawlResult {
	return &CrawlResult{
		Words:   mapKeys(wordSet),
		Secrets: secrets,
		Context: buildContextFromPages(pageContexts, defaultContextLimit(opts)),
		URL:     proto + "://" + addr,
		Pages:   filesProcessed,
	}
}
