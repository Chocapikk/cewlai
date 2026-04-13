package crawler

import (
	"net/url"
	"testing"
)

func TestCaptureURLComponents_Paths(t *testing.T) {
	reqURL, _ := url.Parse("https://example.com/about/team/index.html")
	baseURL, _ := url.Parse("https://example.com")
	wordSet := make(map[string]struct{})

	captureURLComponents(reqURL, baseURL, wordSet, CrawlOptions{CapturePaths: true})

	for _, expected := range []string{"about", "team"} {
		if _, ok := wordSet[expected]; !ok {
			t.Errorf("expected path component %q, got %v", expected, wordSet)
		}
	}
	// index.html should have .html stripped
	if _, ok := wordSet["index.html"]; ok {
		t.Error("expected file extension to be stripped from path component")
	}
}

func TestCaptureURLComponents_Domain(t *testing.T) {
	reqURL, _ := url.Parse("https://example.com/page")
	baseURL, _ := url.Parse("https://example.com")
	wordSet := make(map[string]struct{})

	captureURLComponents(reqURL, baseURL, wordSet, CrawlOptions{CaptureDomain: true})

	if _, ok := wordSet["example"]; !ok {
		t.Errorf("expected domain 'example', got %v", wordSet)
	}
}

func TestCaptureURLComponents_Subdomains(t *testing.T) {
	reqURL, _ := url.Parse("https://blog.shop.example.com/page")
	baseURL, _ := url.Parse("https://example.com")
	wordSet := make(map[string]struct{})

	captureURLComponents(reqURL, baseURL, wordSet, CrawlOptions{CaptureSubdomains: true})

	for _, expected := range []string{"blog", "shop"} {
		if _, ok := wordSet[expected]; !ok {
			t.Errorf("expected subdomain %q, got %v", expected, wordSet)
		}
	}
}
