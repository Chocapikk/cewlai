package crawler

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Chocapikk/cewlai/crawler/parser"
	"github.com/Chocapikk/cewlai/words"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

type httpSource struct {
	url string
}

func (h *httpSource) Crawl(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
	opts.URL = h.url
	return crawlHTTP(ctx, opts)
}

type crawlState struct {
	mu                sync.Mutex
	wordSet           map[string]struct{}
	emailSet          map[string]struct{}
	metaSet           map[string]struct{}
	pageContexts      []string
	secrets           []string
	title             string
	pages             int
	opts              CrawlOptions
	baseURL           *url.URL
	excludeSet        map[string]struct{}
	headerMap         map[string]string
	resourceCollector *colly.Collector
	scanner           *parser.SecretScanner
}

func (s *crawlState) addWords(text string) {
	for _, w := range words.NormalizeAndSplit(text) {
		s.wordSet[w] = struct{}{}
	}
}

func (s *crawlState) addContext(text string) {
	text = strings.TrimSpace(text)
	if text != "" {
		s.pageContexts = append(s.pageContexts, text)
	}
}

func (s *crawlState) buildContext() string {
	return buildContextFromPages(s.pageContexts, defaultContextLimit(s.opts))
}

func (s *crawlState) addEmail(email string) {
	if email = strings.TrimSpace(email); email != "" {
		s.emailSet[strings.ToLower(email)] = struct{}{}
	}
}

func crawlHTTP(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
	if !opts.NoCache {
		ttl := opts.CacheTTL
		if ttl == 0 {
			ttl = time.Hour
		}
		if cached, ok := loadCache(opts.URL, opts.Depth, ttl); ok {
			return cached, nil
		}
	}

	parsed, err := url.Parse(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	s := &crawlState{
		wordSet:    make(map[string]struct{}),
		emailSet:   make(map[string]struct{}),
		metaSet:    make(map[string]struct{}),
		opts:       opts,
		baseURL:    parsed,
		excludeSet: toSet(opts.ExcludePaths),
		headerMap:  parseHeaders(opts.Headers),
	}
	if opts.ExtractSecrets {
		s.scanner = parser.NewSecretScanner()
	}

	c, err := s.buildCollector()
	if err != nil {
		return nil, err
	}

	rc := c.Clone()
	rc.MaxDepth = 0
	s.resourceCollector = rc
	rc.OnResponse(s.onResponse)
	rc.OnError(s.onError)

	s.registerCallbacks(c)

	go func() {
		<-ctx.Done()
		c.AllowedDomains = []string{}
		rc.AllowedDomains = []string{}
	}()

	if err := c.Visit(opts.URL); err != nil {
		return nil, fmt.Errorf("failed to start crawl: %w", err)
	}
	c.Wait()
	rc.Wait()
	fmt.Fprintf(os.Stderr, "\r\033[K")

	ctxText := s.buildContext()

	result := &CrawlResult{
		Words:    mapKeys(s.wordSet),
		Emails:   mapKeys(s.emailSet),
		Metadata: mapKeys(s.metaSet),
		Secrets:  s.secrets,
		Context:  ctxText,
		URL:      opts.URL,
		Title:    s.title,
		Pages:    s.pages,
	}

	if !opts.NoCache {
		saveCache(opts.URL, opts.Depth, result)
	}

	return result, nil
}

func (s *crawlState) buildCollector() (*colly.Collector, error) {
	collectorOpts := []colly.CollectorOption{
		colly.MaxDepth(s.opts.Depth + 1), // colly starts at depth 1, CeWL at depth 0
		colly.UserAgent(s.opts.UserAgent),
	}
	if !s.opts.Offsite {
		host := s.baseURL.Hostname()
		if strings.HasPrefix(host, "www.") {
			collectorOpts = append(collectorOpts, colly.AllowedDomains(host, strings.TrimPrefix(host, "www.")))
		} else {
			collectorOpts = append(collectorOpts, colly.AllowedDomains(host, "www."+host))
		}
	}

	c := colly.NewCollector(collectorOpts...)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if s.opts.ProxyURL != "" {
		proxyParsed, err := url.Parse(s.opts.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyParsed)
	}
	c.WithTransport(transport)

	threads := s.opts.Threads
	if threads < 1 {
		threads = 2
	}
	_ = c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: threads})

	return c, nil
}

func (s *crawlState) registerCallbacks(c *colly.Collector) {
	c.OnRequest(s.onRequest)
	c.OnResponse(s.onResponse)
	c.OnError(s.onError)
}

func (s *crawlState) onRequest(r *colly.Request) {
	s.mu.Lock()
	if s.opts.MaxPages > 0 && s.pages >= s.opts.MaxPages {
		s.mu.Unlock()
		r.Abort()
		return
	}
	s.pages++
	if s.opts.Verbose {
		log.Printf("Crawling: %s", r.URL.String())
	} else {
		fmt.Fprintf(os.Stderr, "\r[*] Crawling: %d pages discovered", s.pages)
	}
	s.mu.Unlock()

	if _, excluded := s.excludeSet[r.URL.Path]; excluded {
		r.Abort()
		return
	}

	if s.opts.AuthType == "basic" && s.opts.AuthUser != "" {
		creds := base64.StdEncoding.EncodeToString([]byte(s.opts.AuthUser + ":" + s.opts.AuthPass))
		r.Headers.Set("Authorization", "Basic "+creds)
	}
	for k, v := range s.headerMap {
		r.Headers.Set(k, v)
	}
}

func (s *crawlState) onResponse(r *colly.Response) {
	contentType := r.Headers.Get("Content-Type")
	reqURL := r.Request.URL.String()

	if s.opts.DumpDir != "" && len(r.Body) > 0 {
		urlPath := r.Request.URL.Path
		if urlPath == "" || urlPath == "/" {
			urlPath = "/index.html"
		}
		dumpFile(s.opts.DumpDir, r.Request.URL.Host+urlPath, r.Body)
	}

	if parser.MatchType(contentType, reqURL, []string{"text/html"}, []string{".html", ".htm"}) {
		s.processHTML(r)
	}

	s.mu.Lock()
	for _, p := range parser.Parsers {
		if parser.MatchType(contentType, reqURL, p.Types, p.Exts) {
			p.Parse(r.Body, s.wordSet)
		}
	}
	s.mu.Unlock()

	if s.opts.ExtractMeta {
		s.processMeta(r.Body, reqURL)
	}

	if s.opts.CapturePaths || s.opts.CaptureSubdomains || s.opts.CaptureDomain {
		s.mu.Lock()
		parser.CaptureURLComponents(r.Request.URL, s.baseURL, s.wordSet, s.opts.CapturePaths, s.opts.CaptureSubdomains, s.opts.CaptureDomain)
		s.mu.Unlock()
	}

	if s.scanner != nil && len(r.Body) > 0 {
		findings := s.scanner.Scan(r.Body, reqURL)
		if len(findings) > 0 {
			s.mu.Lock()
			for _, f := range findings {
				s.secrets = append(s.secrets, fmt.Sprintf("[%s] %s (source: %s)", f.DetectorName, f.Raw, f.Source))
			}
			s.mu.Unlock()
		}
	}
}

func (s *crawlState) processHTML(r *colly.Response) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
	if err != nil {
		return
	}

	// Extract from inline JS before removing script tags
	doc.Find("script").Each(func(_ int, sel *goquery.Selection) {
		if js := sel.Text(); js != "" {
			s.mu.Lock()
			urls := parser.ExtractFromJS([]byte(js), s.wordSet)
			s.mu.Unlock()
			for _, u := range urls {
				_ = r.Request.Visit(r.Request.AbsoluteURL(u))
			}
		}
	})

	doc.Find("script, style").Remove()

	s.mu.Lock()
	if s.title == "" {
		if t := parser.ExtractTitle(doc); t != "" {
			s.title = t
		}
	}
	parser.ExtractAttrs(doc, s.addWords, s.addContext)
	parser.ExtractComments(doc, s.addWords)
	parser.ExtractBodyText(doc, s.addWords, s.addContext)
	s.mu.Unlock()

	if s.opts.ExtractEmails {
		s.mu.Lock()
		parser.ExtractEmails(doc, s.addEmail)
		s.mu.Unlock()
	}

	parser.FollowLinks(doc, func(href string) {
		_ = r.Request.Visit(r.Request.AbsoluteURL(href))
	})

	parser.FollowResources(doc, func(src string) {
		_ = s.resourceCollector.Visit(r.Request.AbsoluteURL(src))
	})
}

func (s *crawlState) processMeta(body []byte, reqURL string) {
	if parser.PdfExt.MatchString(reqURL) {
		parser.ExtractPDFMetadata(body, &s.mu, s.metaSet, s.opts.Verbose, reqURL)
	} else if parser.DocumentExts.MatchString(reqURL) {
		parser.ExtractOfficeMetadata(body, &s.mu, s.metaSet, s.opts.Verbose, reqURL)
	}
}

func (s *crawlState) onError(r *colly.Response, err error) {
	if s.opts.Verbose {
		log.Printf("Error crawling %s: %v", r.Request.URL, err)
	}
	if len(r.Body) > 0 {
		s.onResponse(r)
	}
}

func parseHeaders(headers []string) map[string]string {
	m := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m
}

func toSet(items []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, item := range items {
		m[item] = struct{}{}
	}
	return m
}
