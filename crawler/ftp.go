package crawler

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

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

	downloader := ftpPoolDownloader(addr, user, pass, opts.Threads)
	pageContexts, secrets, processed := processFiles("FTP", files, wordSet, opts, downloader)

	return buildFileResult("ftp", addr, wordSet, pageContexts, secrets, processed, opts), nil
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

func ftpListAll(conn *ftp.ServerConn, startPath string) ([]discoveredFile, map[string]struct{}) {
	wordSet := make(map[string]struct{})
	var files []discoveredFile
	dirs := []string{startPath}

	for len(dirs) > 0 {
		dir := dirs[0]
		dirs = dirs[1:]

		entries, err := conn.List(dir)
		if err != nil {
			continue
		}

		newDirs, newFiles := ftpClassifyEntries(entries, dir, wordSet)
		dirs = append(dirs, newDirs...)
		files = append(files, newFiles...)
	}

	return files, wordSet
}

func ftpClassifyEntries(entries []*ftp.Entry, dir string, wordSet map[string]struct{}) ([]string, []discoveredFile) {
	var dirs []string
	var files []discoveredFile

	for _, entry := range entries {
		if entry.Name == "." || entry.Name == ".." {
			continue
		}
		addNamesToWordSet(entry.Name, wordSet)
		path := dir + "/" + entry.Name
		if entry.Type == ftp.EntryTypeFolder {
			dirs = append(dirs, path)
			continue
		}
		files = append(files, discoveredFile{path: path, name: entry.Name})
	}

	return dirs, files
}

func ftpPoolDownloader(addr, user, pass string, threads int) func(discoveredFile) ([]byte, error) {
	if threads < 1 {
		threads = 2
	}
	pool := make(chan *ftp.ServerConn, threads)

	return func(f discoveredFile) ([]byte, error) {
		conn := ftpGetConn(pool, addr, user, pass)
		if conn == nil {
			return nil, fmt.Errorf("FTP pool exhausted")
		}
		resp, err := conn.Retr(f.path)
		if err != nil {
			ftpPutConn(pool, conn)
			return nil, err
		}
		data, err := io.ReadAll(resp)
		_ = resp.Close()
		ftpPutConn(pool, conn)
		return data, err
	}
}

func ftpGetConn(pool chan *ftp.ServerConn, addr, user, pass string) *ftp.ServerConn {
	select {
	case c := <-pool:
		return c
	default:
		c, err := ftpConnect(addr, user, pass)
		if err != nil {
			return nil
		}
		return c
	}
}

func ftpPutConn(pool chan *ftp.ServerConn, c *ftp.ServerConn) {
	select {
	case pool <- c:
	default:
		_ = c.Quit()
	}
}
