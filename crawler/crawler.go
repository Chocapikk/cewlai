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

	"golang.org/x/net/html"

	"github.com/Chocapikk/cewlai/words"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
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

type crawlState struct {
	mu         sync.Mutex
	wordSet            map[string]struct{}
	emailSet           map[string]struct{}
	metaSet            map[string]struct{}
	contextBuf         strings.Builder
	title              string
	pages              int
	opts               CrawlOptions
	baseURL            *url.URL
	excludeSet         map[string]struct{}
	headerMap          map[string]string
	resourceCollector  *colly.Collector
}

func (s *crawlState) addWords(text string) {
	for _, w := range words.NormalizeAndSplit(text) {
		s.wordSet[w] = struct{}{}
	}
}

func (s *crawlState) addContext(text string) {
	if s.contextBuf.Len() < s.contextLimit() {
		s.contextBuf.WriteString(text + " ")
	}
}

func (s *crawlState) contextLimit() int {
	if s.opts.MaxContext > 0 {
		return s.opts.MaxContext
	}
	return 4000
}

func (s *crawlState) addEmail(email string) {
	if email = strings.TrimSpace(email); email != "" {
		s.emailSet[strings.ToLower(email)] = struct{}{}
	}
}

func Crawl(ctx context.Context, opts CrawlOptions) (*CrawlResult, error) {
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

	ctxText := s.contextBuf.String()
	limit := s.contextLimit()
	if len(ctxText) > limit {
		ctxText = ctxText[:limit]
	}

	return &CrawlResult{
		Words:    mapKeys(s.wordSet),
		Emails:   mapKeys(s.emailSet),
		Metadata: mapKeys(s.metaSet),
		Context:  ctxText,
		URL:      opts.URL,
		Title:    s.title,
		Pages:    s.pages,
	}, nil
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

type contentParser struct {
	types []string
	exts  []string
	parse func(body []byte, wordSet map[string]struct{})
}

func matchType(contentType, reqURL string, types []string, exts []string) bool {
	for _, t := range types {
		if strings.Contains(contentType, t) {
			return true
		}
	}
	lower := strings.ToLower(reqURL)
	for _, ext := range exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

var parsers = []contentParser{
	{[]string{"javascript", "ecmascript"}, []string{".js", ".mjs"}, func(body []byte, wordSet map[string]struct{}) {
		extractFromJS(body, wordSet)
	}},
	{[]string{"xml", "svg"}, []string{".xml", ".svg", ".rss", ".atom", ".sitemap"}, extractFromXML},
	{[]string{"json"}, []string{".json", ".webmanifest"}, extractFromJSON},
	{[]string{"css"}, []string{".css"}, extractFromCSS},
	{[]string{"text/vtt", "subrip"}, []string{".vtt", ".srt"}, extractSubtitles},
	{[]string{"audio", "video"}, []string{".mp3", ".mp4", ".ogg", ".flac", ".wav", ".m4a", ".webm"}, extractMediaMetadata},
}

func (s *crawlState) onResponse(r *colly.Response) {
	contentType := r.Headers.Get("Content-Type")
	reqURL := r.Request.URL.String()

	if matchType(contentType, reqURL, []string{"text/html"}, []string{".html", ".htm"}) {
		s.processHTML(r)
	}

	s.mu.Lock()
	for _, p := range parsers {
		if matchType(contentType, reqURL, p.types, p.exts) {
			p.parse(r.Body, s.wordSet)
		}
	}
	s.mu.Unlock()

	if s.opts.ExtractMeta {
		s.processMeta(r.Body, reqURL)
	}

	if s.opts.CapturePaths || s.opts.CaptureSubdomains || s.opts.CaptureDomain {
		s.mu.Lock()
		captureURLComponents(r.Request.URL, s.baseURL, s.wordSet, s.opts)
		s.mu.Unlock()
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
			urls := extractFromJS([]byte(js), s.wordSet)
			s.mu.Unlock()
			for _, u := range urls {
				_ = r.Request.Visit(r.Request.AbsoluteURL(u))
			}
		}
	})

	doc.Find("script, style").Remove()

	s.mu.Lock()
	s.extractTitle(doc)
	s.extractAttrs(doc)
	s.extractComments(doc)
	s.extractBodyText(doc)
	s.mu.Unlock()

	if s.opts.ExtractEmails {
		s.extractEmails(doc, r)
	}

	s.followLinks(doc, r)
}

func (s *crawlState) extractTitle(doc *goquery.Document) {
	if s.title != "" {
		return
	}
	if t := strings.TrimSpace(doc.Find("title").First().Text()); t != "" {
		s.title = t
	}
}

var wordAttrs = []string{
	"alt", "title", "placeholder", "aria-label", "aria-description",
	"data-title", "data-name", "data-label", "data-value",
	"content", "value", "label", "summary",
}

func (s *crawlState) extractAttrs(doc *goquery.Document) {
	doc.Find("meta[name=description], meta[name=keywords]").Each(func(_ int, sel *goquery.Selection) {
		if content, exists := sel.Attr("content"); exists && content != "" {
			s.addWords(content)
			s.addContext(content)
		}
	})

	doc.Find("*").Each(func(_ int, sel *goquery.Selection) {
		for _, attr := range wordAttrs {
			if val, exists := sel.Attr(attr); exists && val != "" {
				s.addWords(val)
			}
		}
	})
}

func (s *crawlState) extractComments(doc *goquery.Document) {
	doc.Contents().Each(func(_ int, sel *goquery.Selection) {
		for _, node := range sel.Nodes {
			extractCommentsFromNode(node, s)
		}
	})
}

func extractCommentsFromNode(node *html.Node, s *crawlState) {
	if node.Type == html.CommentNode {
		s.addWords(strings.TrimSpace(node.Data))
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		extractCommentsFromNode(child, s)
	}
}

func (s *crawlState) extractBodyText(doc *goquery.Document) {
	doc.Find("*").Each(func(_ int, sel *goquery.Selection) {
		sel.PrependHtml(" ")
	})
	bodyText := doc.Text()
	s.addWords(bodyText)
	s.addContext(bodyText)
}

func (s *crawlState) extractEmails(doc *goquery.Document, r *colly.Response) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc.Find("a[href^='mailto:']").Each(func(_ int, sel *goquery.Selection) {
		if href, exists := sel.Attr("href"); exists {
			email := strings.TrimPrefix(href, "mailto:")
			email = strings.Split(email, "?")[0]
			s.addEmail(email)
		}
	})

	bodyText := doc.Text()
	for _, e := range extractEmailsFromText(bodyText) {
		s.emailSet[e] = struct{}{}
	}
}

func (s *crawlState) followLinks(doc *goquery.Document, r *colly.Response) {
	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		if val, exists := sel.Attr("href"); exists {
			_ = r.Request.Visit(r.Request.AbsoluteURL(val))
		}
	})

	resources := []struct {
		query string
		attr  string
	}{
		{"script[src]", "src"},
		{"link[href]", "href"},
		{"img[src]", "src"},
		{"iframe[src]", "src"},
		{"source[src]", "src"},
		{"video[src]", "src"},
		{"audio[src]", "src"},
		{"track[src]", "src"},
	}

	for _, res := range resources {
		doc.Find(res.query).Each(func(_ int, sel *goquery.Selection) {
			if val, exists := sel.Attr(res.attr); exists {
				_ = s.resourceCollector.Visit(r.Request.AbsoluteURL(val))
			}
		})
	}
}

func (s *crawlState) processMeta(body []byte, reqURL string) {
	if pdfExt.MatchString(reqURL) {
		extractPDFMetadata(body, &s.mu, s.metaSet, s.opts.Verbose, reqURL)
	} else if documentExts.MatchString(reqURL) {
		extractOfficeMetadata(body, &s.mu, s.metaSet, s.opts.Verbose, reqURL)
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

func mapKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
