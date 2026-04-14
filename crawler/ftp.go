package crawler

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Chocapikk/cewlai/words"
	"github.com/jlaffaye/ftp"
)

func crawlFTP(addr, user, pass string, opts CrawlOptions) (*CrawlResult, error) {
	if user == "" {
		user = "anonymous"
	}
	if pass == "" {
		pass = "anonymous@"
	}
	if !strings.Contains(addr, ":") {
		addr += ":21"
	}

	conn, err := ftp.Dial(addr, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return nil, fmt.Errorf("FTP connect failed: %w", err)
	}
	defer func() { _ = conn.Quit() }()

	if err := conn.Login(user, pass); err != nil {
		return nil, fmt.Errorf("FTP login failed: %w", err)
	}

	wordSet := make(map[string]struct{})
	var pageContexts []string
	filesProcessed := 0

	var walk func(dir string)
	walk = func(dir string) {
		entries, err := conn.List(dir)
		if err != nil {
			return
		}

		for _, entry := range entries {
			if entry.Name == "." || entry.Name == ".." {
				continue
			}

			path := dir + "/" + entry.Name

			for _, w := range words.NormalizeAndSplit(entry.Name) {
				wordSet[w] = struct{}{}
			}

			if entry.Type == ftp.EntryTypeFolder {
				walk(path)
				continue
			}

			body, err := downloadFTPFile(conn, path)
			if err != nil || len(body) == 0 {
				continue
			}

			ext := strings.ToLower(filepath.Ext(entry.Name))
			processFileContent(ext, body, wordSet, &pageContexts)
			filesProcessed++
			fmt.Fprintf(os.Stderr, "\r[*] FTP: %d files processed", filesProcessed)
		}
	}

	walk("/")
	fmt.Fprintf(os.Stderr, "\r\033[K")

	ctxText := buildContextFromPages(pageContexts, contextLimitFromOpts(opts))

	return &CrawlResult{
		Words:   mapKeys(wordSet),
		Context: ctxText,
		URL:     "ftp://" + addr,
		Pages:   filesProcessed,
	}, nil
}

func downloadFTPFile(conn *ftp.ServerConn, path string) ([]byte, error) {
	resp, err := conn.Retr(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Close() }()
	return io.ReadAll(resp)
}

func processFileContent(ext string, body []byte, wordSet map[string]struct{}, pageContexts *[]string) {
	// Text-based files: extract words + add to AI context
	switch ext {
	case ".txt", ".md", ".csv", ".log", ".conf", ".cfg", ".ini", ".yml", ".yaml":
		extractTextContent(body, wordSet, pageContexts)
		return
	}

	// Try the shared parsers table (same as HTTP crawler)
	for _, p := range parsers {
		for _, e := range p.exts {
			if ext == e {
				p.parse(body, wordSet)
				return
			}
		}
	}

	// PDF/Office metadata (need mutex)
	var mu sync.Mutex
	switch {
	case ext == ".pdf":
		extractPDFMetadata(body, &mu, wordSet, false, "")
	case ext == ".docx" || ext == ".xlsx" || ext == ".pptx" || ext == ".dotx" || ext == ".potx" || ext == ".ppsx":
		extractOfficeMetadata(body, &mu, wordSet, false, "")
	}
}

func extractTextContent(body []byte, wordSet map[string]struct{}, pageContexts *[]string) {
	text := string(body)
	for _, w := range words.NormalizeAndSplit(text) {
		wordSet[w] = struct{}{}
	}
	if trimmed := strings.TrimSpace(text); trimmed != "" {
		*pageContexts = append(*pageContexts, trimmed)
	}
}

func contextLimitFromOpts(opts CrawlOptions) int {
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
