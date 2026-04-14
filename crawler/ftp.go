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
	user, pass := "", ""
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
	startPath := f.parsed.Path
	if startPath == "" {
		startPath = "/"
	}
	return crawlFTP(addr, user, pass, startPath, opts)
}

type ftpFile struct {
	path string
	name string
}

func crawlFTP(addr, user, pass, startPath string, opts CrawlOptions) (*CrawlResult, error) {
	if user == "" {
		user = "anonymous"
	}
	if pass == "" {
		pass = "anonymous@"
	}
	if !strings.Contains(addr, ":") {
		addr += ":21"
	}

	conn, err := ftpConnect(addr, user, pass)
	if err != nil {
		return nil, err
	}

	files, wordSet := ftpListAll(conn, startPath)
	_ = conn.Quit()

	var mu sync.Mutex
	var pageContexts []string
	var filesProcessed int32
	total := len(files)

	ftpProcessFiles(addr, user, pass, files, opts.Threads, func(f ftpFile, body []byte) {
		ext := strings.ToLower(filepath.Ext(f.name))
		mu.Lock()
		parser.ParseByExtension(ext, body, wordSet, &pageContexts)
		filesProcessed++
		mu.Unlock()
		fmt.Fprintf(os.Stderr, "\r[*] FTP: %d/%d files processed", filesProcessed, total)
	})

	fmt.Fprintf(os.Stderr, "\r\033[K")

	return &CrawlResult{
		Words:   mapKeys(wordSet),
		Context: buildContextFromPages(pageContexts, defaultContextLimit(opts)),
		URL:     "ftp://" + addr,
		Pages:   int(filesProcessed),
	}, nil
}

func ftpConnect(addr, user, pass string) (*ftp.ServerConn, error) {
	conn, err := ftp.Dial(addr, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return nil, fmt.Errorf("FTP connect failed: %w", err)
	}
	if err := conn.Login(user, pass); err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("FTP login failed: %w", err)
	}
	return conn, nil
}

func ftpListAll(conn *ftp.ServerConn, startPath string) ([]ftpFile, map[string]struct{}) {
	wordSet := make(map[string]struct{})
	var files []ftpFile
	dirs := []string{startPath}

	for len(dirs) > 0 {
		dir := dirs[0]
		dirs = dirs[1:]

		entries, err := conn.List(dir)
		if err != nil {
			continue
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
				dirs = append(dirs, path)
			} else {
				files = append(files, ftpFile{path: path, name: entry.Name})
			}
		}
	}

	return files, wordSet
}

func ftpProcessFiles(addr, user, pass string, files []ftpFile, threads int, process func(ftpFile, []byte)) {
	if threads < 1 {
		threads = 2
	}

	work := make(chan ftpFile)
	var wg sync.WaitGroup

	for range threads {
		wg.Add(1)
		go ftpWorker(addr, user, pass, work, &wg, process)
	}

	for _, f := range files {
		work <- f
	}
	close(work)
	wg.Wait()
}

func ftpWorker(addr, user, pass string, work <-chan ftpFile, wg *sync.WaitGroup, process func(ftpFile, []byte)) {
	defer wg.Done()

	conn, err := ftpConnect(addr, user, pass)
	if err != nil {
		return
	}
	defer func() { _ = conn.Quit() }()

	for f := range work {
		body, err := ftpDownload(conn, f.path)
		if err != nil || len(body) == 0 {
			continue
		}
		process(f, body)
	}
}

func ftpDownload(conn *ftp.ServerConn, path string) ([]byte, error) {
	resp, err := conn.Retr(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Close() }()
	return io.ReadAll(resp)
}
