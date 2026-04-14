package crawler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type CrawlOptions struct {
	URL               string
	Depth             int
	UserAgent         string
	Verbose           bool
	Offsite           bool
	WithNumbers       bool
	ExtractEmails     bool
	ExtractMeta       bool
	CapturePaths      bool
	CaptureSubdomains bool
	CaptureDomain     bool
	MaxPages          int
	Threads           int
	MaxContext        int
	NoCache           bool
	CacheTTL          time.Duration
	MaxFiles          int
	ProxyURL          string
	AuthType          string
	AuthUser          string
	AuthPass          string
	Headers           []string
	ExcludePaths      []string
}

type CrawlResult struct {
	Words    []string
	Emails   []string
	Metadata []string
	Context  string
	URL      string
	Title    string
	Pages    int
}

type Source interface {
	Crawl(ctx context.Context, opts CrawlOptions) (*CrawlResult, error)
}

func NewSource(targetURL string) (Source, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "ftp":
		return &ftpSource{parsed: parsed}, nil
	case "smb":
		return &smbSource{parsed: parsed}, nil
	case "sftp":
		return &sftpSource{parsed: parsed}, nil
	case "http", "https", "":
		return &httpSource{url: targetURL}, nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %s (supported: http, https, ftp, sftp, smb)", parsed.Scheme)
	}
}

func Crawl(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
	src, err := NewSource(opts.URL)
	if err != nil {
		return nil, err
	}
	return src.Crawl(ctx, opts)
}
