package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/Chocapikk/cewlai/ai"
	"github.com/Chocapikk/cewlai/crawler"
	"github.com/Chocapikk/cewlai/words"
	"github.com/alecthomas/kong"
	kongcompletion "github.com/jotaen/kong-completion"
)

var version = "dev"

func getBanner() string {
	return fmt.Sprintf(`
  ____   __        ___          _    ___
 / ___|__\ \      / / |        / \  |_ _|
| |   / _ \ \ /\ / /| |       / _ \  | |
| |__|  __/\ V  V / | |___   / ___ \ | |
 \____\___| \_/\_/  |_____| /_/   \_\___|
      AI-Powered Wordlist Generator
          & Target Recon Tool
  Created by @Chocapikk | %s
  Thanks to @stlthr4k3r for the original idea
`, version)
}

type CLI struct {
	// Target
	Url       string `short:"u" help:"Target URL to crawl" name:"url" group:"Target"`
	Depth     int    `short:"d" default:"2" help:"Crawl depth" group:"Target"`
	Output    string `short:"o" help:"Output file (default: stdout)" group:"Target"`
	Verbose   bool   `short:"v" help:"Verbose output" group:"Target"`
	Version   bool   `help:"Print version and exit" group:"Target"`
	Update    bool   `help:"Self-update to latest release" group:"Target"`

	// Crawling
	UserAgent         string   `default:"cewlai/1.0" help:"User agent for crawler" name:"user-agent" group:"Crawling"`
	Offsite           bool     `help:"Follow offsite links" group:"Crawling"`
	Proxy             string   `help:"HTTP proxy URL" group:"Crawling"`
	AuthType          string   `help:"Auth type: basic" name:"auth-type" group:"Crawling"`
	AuthUser          string   `help:"Auth username" name:"auth-user" group:"Crawling"`
	AuthPass          string   `help:"Auth password" name:"auth-pass" group:"Crawling"`
	Header            []string `help:"Custom header (repeatable, Key: Value)" group:"Crawling"`
	Exclude           string   `help:"File with paths to exclude" group:"Crawling"`
	MaxPages          int      `default:"0" help:"Maximum pages to crawl (0 = no limit)" name:"max-pages" group:"Crawling"`
	MaxFiles          int      `default:"0" help:"Maximum files to process for FTP/SFTP/SMB (0 = no limit)" name:"max-files" group:"Crawling"`
	Threads           int      `short:"t" default:"2" help:"Number of concurrent crawl threads" group:"Crawling"`
	NoCache           bool     `help:"Disable crawl cache" name:"no-cache" group:"Crawling"`
	CacheTTL          int      `default:"60" help:"Cache TTL in minutes" name:"cache-ttl" group:"Crawling"`
	Dump              string   `help:"Dump all crawled files to directory" group:"Crawling"`

	// Extraction
	Email             bool     `short:"e" help:"Extract email addresses" group:"Extraction"`
	EmailFile         string   `help:"Write emails to file" name:"email-file" group:"Extraction"`
	Meta              bool     `short:"a" help:"Extract document metadata" group:"Extraction"`
	MetaFile          string   `help:"Write metadata to file" name:"meta-file" group:"Extraction"`
	Secrets           bool     `short:"s" help:"Extract secrets (API keys, tokens, passwords) via trufflehog detectors" group:"Extraction"`
	SecretsFile       string   `help:"Write secrets to file" name:"secrets-file" group:"Extraction"`
	CapturePaths      bool     `help:"Add URL path components to wordlist" name:"capture-paths" group:"Extraction"`
	CaptureSubdomains bool     `help:"Add subdomains to wordlist" name:"capture-subdomains" group:"Extraction"`
	CaptureDomain     bool     `help:"Add domain to wordlist" name:"capture-domain" group:"Extraction"`

	// Words
	MinWordLength int    `default:"3" help:"Minimum word length" name:"min-word-length" group:"Words"`
	MaxWordLength int    `default:"0" help:"Maximum word length (0 = no limit)" name:"max-word-length" group:"Words"`
	Lowercase     bool   `help:"Lowercase all words" group:"Words"`
	WithNumbers   bool   `default:"true" help:"Include words with numbers" name:"with-numbers" group:"Words"`
	Count         bool   `help:"Show word frequency count" group:"Words"`
	Groups        int    `short:"g" default:"0" help:"Generate word groups of N" group:"Words"`
	Mutate        bool   `help:"Generate word mutations (leet, reverse, suffixes like CUPP)" group:"Words"`
	MutateConfig  string `help:"Custom mutation config file (JSON)" name:"mutate-config" group:"Words"`

	// AI
	AI         bool   `help:"Enable AI enrichment" group:"AI"`
	Provider   string `short:"p" help:"AI provider: anthropic, openai, groq, openrouter, cerebras, huggingface" group:"AI"`
	Model      string `short:"m" help:"Model name or shorthand" group:"AI"`
	APIKey     string `help:"API key (or use env vars)" name:"api-key" group:"AI"`
	BaseURL    string `help:"Custom API base URL for OpenAI-compatible endpoints" name:"base-url" group:"AI"`
	ListModels bool   `help:"List available models for the selected provider" name:"list-models" group:"AI"`
	Mode       string `default:"default" help:"AI prompt mode: default, passwords, dirs, subdomains, geo" group:"AI"`
	Prompt     string `help:"Custom AI system prompt (overrides --mode)" group:"AI"`
	AIWords    int    `default:"200" help:"Number of AI-generated words" name:"ai-words" group:"AI"`
	AIContext  int    `default:"4000" help:"Max characters of context sent to LLM" name:"ai-context" group:"AI"`

	// Shell completion
	Completion kongcompletion.Completion `cmd:"" help:"Output shell completion code (bash, zsh, fish)" hidden:""`
}

func main() {
	var cli CLI
	parser := kong.Must(&cli,
		kong.Name("cewlai"),
		kong.Description("AI-Powered Wordlist Generator & Target Recon Tool"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{NoAppSummary: true}),
	)
	kongcompletion.Register(parser)
	kongCtx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	if strings.HasPrefix(kongCtx.Command(), "completion") {
		_ = kongCtx.Run()
		return
	}

	if cli.Version {
		fmt.Println("cewlai " + version)
		return
	}

	if cli.Update {
		selfUpdate()
		return
	}

	if cli.ListModels {
		ai.ListModels(cli.Provider, cli.APIKey, cli.BaseURL)
		return
	}

	_, _ = fmt.Fprintln(os.Stderr, colorize(cyan, getBanner()))

	verboseMode = cli.Verbose

	targetURL := cli.Url
	if targetURL == "" {
		logFatal("-u (URL) is required")
	}

	var excludePaths []string
	if cli.Exclude != "" {
		excludePaths = readLines(cli.Exclude)
	}

	opts := crawler.CrawlOptions{
		URL:               targetURL,
		Depth:             cli.Depth,
		UserAgent:         cli.UserAgent,
		Verbose:           cli.Verbose,
		Offsite:           cli.Offsite,
		WithNumbers:       cli.WithNumbers,
		ExtractEmails:     cli.Email,
		ExtractMeta:       cli.Meta,
		CapturePaths:      cli.CapturePaths,
		CaptureSubdomains: cli.CaptureSubdomains,
		CaptureDomain:     cli.CaptureDomain,
		MaxPages:          cli.MaxPages,
		MaxContext:        cli.AIContext,
		Threads:           cli.Threads,
		ProxyURL:          cli.Proxy,
		AuthType:          cli.AuthType,
		AuthUser:          cli.AuthUser,
		AuthPass:          cli.AuthPass,
		Headers:           cli.Header,
		ExcludePaths:      excludePaths,
		NoCache:           cli.NoCache,
		CacheTTL:          time.Duration(cli.CacheTTL) * time.Minute,
		MaxFiles:          cli.MaxFiles,
		ExtractSecrets:    cli.Secrets,
		DumpDir:           cli.Dump,
	}

	logInfo("Starting crawl on %s (depth: %d)", targetURL, cli.Depth)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		_, _ = fmt.Fprintf(os.Stderr, "\r\033[K")
		logResult("Interrupted, saving partial results...")
		cancel()
	}()

	start := time.Now()
	result, err := crawler.Crawl(ctx, opts)
	if err != nil {
		logFatal("Crawl failed: %v", err)
	}

	logResult("Crawled %d pages, extracted %d raw words in %.1fs", result.Pages, len(result.Words), time.Since(start).Seconds())

	crawlWords := words.FilterWords(result.Words, cli.MinWordLength, cli.MaxWordLength, cli.WithNumbers)
	if cli.Lowercase {
		crawlWords = words.LowercaseWords(crawlWords)
	}

	var aiWords []string
	if cli.AI {
		aiWords = enrichWithAI(cli, result)
	}

	merged := words.DeduplicateWords(crawlWords, aiWords)

	if cli.Mutate {
		cfg := words.DefaultMutateConfig()
		if cli.MutateConfig != "" {
			var err error
			cfg, err = words.LoadMutateConfig(cli.MutateConfig)
			if err != nil {
				logFatal("Failed to load mutate config: %v", err)
			}
		}
		logInfo("Generating mutations...")
		merged = words.MutateWords(merged, cfg)
		logResult("Mutated to %d words", len(merged))
	}

	final := words.DeduplicateWords(merged)
	logResult("Final wordlist: %d unique words", len(final))

	writeWordlist(final, cli.Output, cli.Count, cli.Groups)
	writeExtras(result, cli)
}

func enrichWithAI(cli CLI, result *crawler.CrawlResult) []string {
	if cli.Provider == "" {
		logFatal("-p (provider) is required with --ai")
	}

	apiKey := cli.APIKey
	if apiKey == "" {
		switch strings.ToLower(cli.Provider) {
		case "anthropic":
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		case "openai":
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
	}

	p, err := ai.NewAIProvider(cli.Provider, apiKey, cli.Model, cli.BaseURL)
	if err != nil {
		logFatal("AI provider error: %v", err)
	}

	target := cli.AIWords
	maxTokens := ai.MaxTokensForWords(target)
	logInfo("Sending context to %s for enrichment (mode: %s, target: %d words)...", cli.Provider, cli.Mode, target)

	seen := make(map[string]struct{})
	var aiWords []string
	attempt := 0

	for len(aiWords) < target {
		attempt++
		remaining := target - len(aiWords)

		prompt := ai.ResolvePrompt(cli.Mode, cli.Prompt, remaining)
		batch, err := p.GenerateWords(context.Background(), result, prompt, maxTokens)
		if err != nil {
			logError("AI request failed: %v", err)
			break
		}

		batch = words.FilterWords(batch, cli.MinWordLength, cli.MaxWordLength, cli.WithNumbers)
		if cli.Lowercase {
			batch = words.LowercaseWords(batch)
		}

		added := 0
		for _, w := range batch {
			if _, exists := seen[w]; exists {
				continue
			}
			seen[w] = struct{}{}
			aiWords = append(aiWords, w)
			added++
		}

		logInfo("Attempt %d: got %d/%d words (+%d new)", attempt, len(aiWords), target, added)

		if added == 0 {
			logInfo("AI exhausted context, stopping")
			break
		}
	}

	if len(aiWords) > target {
		aiWords = aiWords[:target]
	}

	logResult("AI generated %d words", len(aiWords))
	return aiWords
}

func writeWordlist(wordList []string, output string, showCount bool, groupSize int) {
	w := openOutput(output)
	if w != os.Stdout {
		defer func() { _ = w.Close() }()
	}

	for _, line := range formatWordlist(wordList, showCount, groupSize) {
		_, _ = fmt.Fprintln(w, line)
	}
}

func formatWordlist(wordList []string, showCount bool, groupSize int) []string {
	if groupSize > 0 {
		return words.GenerateGroups(wordList, groupSize)
	}
	if showCount {
		return words.FormatWithCounts(wordList)
	}
	return wordList
}

func writeExtras(result *crawler.CrawlResult, cli CLI) {
	if len(result.Emails) > 0 {
		sort.Strings(result.Emails)
		logSuccess("Extracted %d emails", len(result.Emails))
		writeLines(result.Emails, cli.EmailFile)
	}

	if len(result.Metadata) > 0 {
		sort.Strings(result.Metadata)
		logSuccess("Extracted %d metadata entries", len(result.Metadata))
		writeLines(result.Metadata, cli.MetaFile)
	}

	if len(result.Secrets) > 0 {
		logSuccess("Found %d secrets", len(result.Secrets))
		writeLines(result.Secrets, cli.SecretsFile)
	}
}

func writeLines(lines []string, path string) {
	if path == "" {
		for _, l := range lines {
			_, _ = fmt.Fprintln(os.Stderr, l)
		}
		return
	}
	f, err := os.Create(path)
	if err != nil {
		logFatal("Failed to create file %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	for _, l := range lines {
		_, _ = fmt.Fprintln(f, l)
	}
}

func openOutput(path string) *os.File {
	if path == "" {
		return os.Stdout
	}
	f, err := os.Create(path)
	if err != nil {
		logFatal("Failed to create output file: %v", err)
	}
	return f
}

func readLines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		logFatal("Failed to open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
