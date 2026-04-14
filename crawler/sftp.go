package crawler

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sftpSource struct {
	parsed *url.URL
}

func (s *sftpSource) Crawl(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
	addr := s.parsed.Hostname() + ":" + sftpPort(s.parsed)
	user, pass := sftpCreds(s.parsed, opts)
	startPath := s.parsed.Path
	if startPath == "" {
		startPath = "/"
	}
	return crawlSFTP(addr, user, pass, startPath, opts)
}

func sftpPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	return "22"
}

func sftpCreds(u *url.URL, opts CrawlOptions) (string, string) {
	user, pass := "", ""
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	if opts.AuthUser != "" {
		user = opts.AuthUser
	}
	if opts.AuthPass != "" {
		pass = opts.AuthPass
	}
	return user, pass
}

func crawlSFTP(addr, user, pass, startPath string, opts CrawlOptions) (*CrawlResult, error) {
	client, err := sftpConnect(addr, user, pass)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()

	files, wordSet := sftpListAll(client, startPath)

	downloader := func(f discoveredFile) ([]byte, error) {
		return sftpDownload(client, f.path)
	}
	pageContexts, processed := processFiles("SFTP", files, wordSet, opts, downloader)

	return buildFileResult("sftp", addr, wordSet, pageContexts, processed, opts), nil
}

func sftpConnect(addr, user, pass string) (*sftp.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH connect failed: %w", err)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("SFTP session failed: %w", err)
	}

	return client, nil
}

func sftpListAll(client *sftp.Client, startPath string) ([]discoveredFile, map[string]struct{}) {
	wordSet := make(map[string]struct{})
	var files []discoveredFile
	dirs := []string{startPath}

	for len(dirs) > 0 {
		dir := dirs[0]
		dirs = dirs[1:]

		entries, err := client.ReadDir(dir)
		if err != nil {
			continue
		}

		newDirs, newFiles := sftpClassifyEntries(entries, dir, wordSet)
		dirs = append(dirs, newDirs...)
		files = append(files, newFiles...)
	}

	return files, wordSet
}

func sftpClassifyEntries(entries []os.FileInfo, dir string, wordSet map[string]struct{}) ([]string, []discoveredFile) {
	var dirs []string
	var files []discoveredFile

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		addNamesToWordSet(name, wordSet)
		path := dir + "/" + name

		if entry.IsDir() {
			dirs = append(dirs, path)
			continue
		}
		if !entry.Mode().IsRegular() || entry.Size() > 50*1024*1024 {
			continue
		}
		files = append(files, discoveredFile{path: path, name: name})
	}

	return dirs, files
}

func sftpDownload(client *sftp.Client, path string) ([]byte, error) {
	f, err := client.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}
