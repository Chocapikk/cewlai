package crawler

import (
	"net/url"
	"regexp"
	"strings"
)

var fileExtRe = regexp.MustCompile(`\.[a-zA-Z0-9]{1,10}$`)

func captureURLComponents(reqURL, baseURL *url.URL, wordSet map[string]struct{}, opts CrawlOptions) {
	if opts.CapturePaths {
		for _, part := range strings.Split(reqURL.Path, "/") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			part = fileExtRe.ReplaceAllString(part, "")
			if part != "" {
				wordSet[part] = struct{}{}
			}
		}
	}

	if opts.CaptureDomain {
		host := baseURL.Hostname()
		domain := strings.Split(host, ".")[0]
		if domain != "" {
			wordSet[domain] = struct{}{}
		}
	}

	if opts.CaptureSubdomains {
		host := reqURL.Hostname()
		baseHost := baseURL.Hostname()
		if strings.HasSuffix(host, baseHost) && host != baseHost {
			sub := strings.TrimSuffix(host, "."+baseHost)
			for _, part := range strings.Split(sub, ".") {
				if part != "" {
					wordSet[part] = struct{}{}
				}
			}
		}
	}
}
