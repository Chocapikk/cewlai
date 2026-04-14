package crawler

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Chocapikk/cewlai/crawler/parser"
	"github.com/Chocapikk/cewlai/words"
	"github.com/jlaffaye/ftp"
)

type ftpSource struct {
	parsed *url.URL
}

func (f *ftpSource) Crawl(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
	addr := f.parsed.Host
	user := ""
	pass := ""
	if f.parsed.User != nil {
		user = f.parsed.User.Username()
		pass, _ = f.parsed.User.Password()
	}
	if opts.AuthUser != "" {
		user = opts.AuthUser
	}
	if opts.AuthPass != "" {
		pass = opts.AuthPass
	}
	return crawlFTP(addr, user, pass, opts)
}

type ftpFile struct {
	path string
	name string
}

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

	// First connection: list all files
	conn, err := ftp.Dial(addr, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return nil, fmt.Errorf("FTP connect failed: %w", err)
	}
	if err := conn.Login(user, pass); err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("FTP login failed: %w", err)
	}

	var files []ftpFile
	wordSet := make(map[string]struct{})

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
			} else {
				files = append(files, ftpFile{path: path, name: entry.Name})
			}
		}
	}
	walk("/")
	_ = conn.Quit()

	// Process files in parallel with worker pool
	var mu sync.Mutex
	var pageContexts []string
	var filesProcessed int32

	threads := opts.Threads
	if threads < 1 {
		threads = 2
	}

	work := make(chan ftpFile)
	var wg sync.WaitGroup

	for range threads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := ftp.Dial(addr, ftp.DialWithTimeout(10*time.Second))
			if err != nil {
				return
			}
			defer func() { _ = c.Quit() }()
			if err := c.Login(user, pass); err != nil {
				return
			}

			for f := range work {
				body, err := downloadFTPFile(c, f.path)
				if err != nil || len(body) == 0 {
					continue
				}

				ext := strings.ToLower(filepath.Ext(f.name))
				mu.Lock()
				parser.ParseByExtension(ext, body, wordSet, &pageContexts)
				filesProcessed++
				mu.Unlock()
				fmt.Fprintf(os.Stderr, "\r[*] FTP: %d/%d files processed", filesProcessed, len(files))
			}
		}()
	}

	for _, f := range files {
		work <- f
	}
	close(work)
	wg.Wait()
	fmt.Fprintf(os.Stderr, "\r\033[K")

	return &CrawlResult{
		Words:   mapKeys(wordSet),
		Context: buildContextFromPages(pageContexts, defaultContextLimit(opts)),
		URL:     "ftp://" + addr,
		Pages:   int(filesProcessed),
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
