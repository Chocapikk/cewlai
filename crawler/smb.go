package crawler

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	smb2 "github.com/cloudsoda/go-smb2"
)

type smbSource struct {
	parsed *url.URL
}

func (s *smbSource) Crawl(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
	addr := s.parsed.Hostname() + ":" + smbPort(s.parsed)
	user, pass := smbCreds(s.parsed, opts)
	share, startDir := smbPath(s.parsed)
	return crawlSMB(ctx, addr, user, pass, share, startDir, opts)
}

func smbPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	return "445"
}

func smbCreds(u *url.URL, opts CrawlOptions) (string, string) {
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

func smbPath(u *url.URL) (string, string) {
	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	share := parts[0]
	startDir := "."
	if len(parts) > 1 && parts[1] != "" {
		startDir = parts[1]
	}
	return share, startDir
}

func crawlSMB(ctx context.Context, addr, user, pass, share, startDir string, opts CrawlOptions) (*CrawlResult, error) {
	session, err := smbConnect(ctx, addr, user, pass)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Logoff() }()

	mount, err := session.Mount(share)
	if err != nil {
		return nil, fmt.Errorf("SMB mount '%s' failed: %w", share, err)
	}
	defer func() { _ = mount.Umount() }()

	files, wordSet := smbListAll(mount, startDir)

	downloader := func(f discoveredFile) ([]byte, error) {
		return smbDownload(mount, f.path)
	}
	pageContexts, processed := processFiles("SMB", files, wordSet, opts, downloader)

	return buildFileResult("smb", addr+"/"+share, wordSet, pageContexts, processed, opts), nil
}

func smbConnect(ctx context.Context, addr, user, pass string) (*smb2.Session, error) {
	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     user,
			Password: pass,
		},
	}
	session, err := d.Dial(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("SMB connect failed: %w", err)
	}
	return session, nil
}

func smbListAll(mount *smb2.Share, startDir string) ([]discoveredFile, map[string]struct{}) {
	wordSet := make(map[string]struct{})
	var files []discoveredFile
	dirs := []string{startDir}

	for len(dirs) > 0 {
		dir := dirs[0]
		dirs = dirs[1:]

		entries, err := mount.ReadDir(dir)
		if err != nil {
			continue
		}

		newDirs, newFiles := smbClassifyEntries(entries, dir, wordSet)
		dirs = append(dirs, newDirs...)
		files = append(files, newFiles...)
	}

	return files, wordSet
}

func smbClassifyEntries(entries []os.FileInfo, dir string, wordSet map[string]struct{}) ([]string, []discoveredFile) {
	var dirs []string
	var files []discoveredFile

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		addNamesToWordSet(name, wordSet)

		var path string
		if dir == "." {
			path = name
		} else {
			path = dir + `\` + name
		}

		if entry.IsDir() {
			dirs = append(dirs, path)
			continue
		}
		files = append(files, discoveredFile{path: path, name: name})
	}

	return dirs, files
}

func smbDownload(mount *smb2.Share, path string) ([]byte, error) {
	f, err := mount.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 50*1024*1024 {
		return nil, fmt.Errorf("skip: too large or not regular")
	}

	return io.ReadAll(f)
}
