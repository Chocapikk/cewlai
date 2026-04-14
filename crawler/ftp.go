package crawler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
			parseByExtension(ext, body, wordSet, &pageContexts)
			filesProcessed++
			fmt.Fprintf(os.Stderr, "\r[*] FTP: %d files processed", filesProcessed)
		}
	}

	walk("/")
	fmt.Fprintf(os.Stderr, "\r\033[K")

	return &CrawlResult{
		Words:   mapKeys(wordSet),
		Context: buildContextFromPages(pageContexts, defaultContextLimit(opts)),
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
