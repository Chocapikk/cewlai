package parser

import (
	"net/url"
	"regexp"
	"strings"
)

var fileExtRe = regexp.MustCompile(`\.[a-zA-Z0-9]{1,10}$`)

func CaptureURLComponents(reqURL, baseURL *url.URL, wordSet map[string]struct{}, capturePaths, captureSubdomains, captureDomain bool) {
	if capturePaths {
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

	if captureDomain {
		host := baseURL.Hostname()
		domain := strings.Split(host, ".")[0]
		if domain != "" {
			wordSet[domain] = struct{}{}
		}
	}

	if captureSubdomains {
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
