package crawler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

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
	case "http", "https", "":
		return &httpSource{url: targetURL}, nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %s (supported: http, https, ftp)", parsed.Scheme)
	}
}

type httpSource struct {
	url string
}

func (h *httpSource) Crawl(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
	opts.URL = h.url
	return crawlHTTP(ctx, opts)
}

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
