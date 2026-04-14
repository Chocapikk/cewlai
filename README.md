# CeWL AI

[![Go CI](https://github.com/Chocapikk/cewlai/actions/workflows/go.yml/badge.svg)](https://github.com/Chocapikk/cewlai/actions/workflows/go.yml)
[![Release](https://img.shields.io/github/v/release/Chocapikk/cewlai)](https://github.com/Chocapikk/cewlai/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/Chocapikk/cewlai)](https://goreportcard.com/report/github.com/Chocapikk/cewlai)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

AI-powered custom wordlist generator. Crawls websites and uses LLMs to generate contextually related words that don't appear on the page - industry jargon, likely passwords, related terminology.

Built on top of the classic [CeWL](https://github.com/digininja/CeWL) concept, rewritten in Go with AI enrichment.

## Install

```bash
go install github.com/Chocapikk/cewlai@latest
```

Or build from source:

```bash
git clone https://github.com/Chocapikk/cewlai.git
cd cewlai
go build -o cewlai .
```

## Usage

Basic crawl (classic CeWL mode):

```bash
cewlai -u https://example.com
```

With AI enrichment:

```bash
cewlai -u https://example.com --ai -p anthropic -m sonnet
```

FTP crawling:

```bash
cewlai -u ftp://anonymous@ftp.example.com
cewlai -u ftp://user:pass@ftp.example.com/share/docs
```

SMB crawling:

```bash
cewlai -u smb://user:pass@192.168.1.10/data
cewlai -u smb://DOMAIN\\user:pass@host/share/path
```

With a free provider:

```bash
export GROQ_API_KEY=gsk_...
cewlai -u https://example.com --ai -p groq
```

> [!TIP]
> Don't know which models are available? List them:
> ```bash
> cewlai --list-models -p groq
> cewlai --list-models -p cerebras
> ```

> [!WARNING]
> No API key? The tool tells you what to set:
> ```
> $ cewlai -u https://example.com --ai -p groq
> [-] AI provider error: no API key for groq. Set GROQ_API_KEY or use --api-key
> ```

Full example:

```bash
cewlai -u https://example.com -d 3 --ai -p anthropic -m haiku \
  --lowercase --email --meta -o wordlist.txt --email-file emails.txt
```

## Flags

```
Usage: cewlai [<url>] [flags]

AI-Powered Wordlist Generator

Arguments:
  [<url>]    Target URL to crawl

Flags:
  -h, --help                       Show context-sensitive help.
  -u, --url=STRING                 Target URL to crawl
  -d, --depth=2                    Crawl depth
  -o, --output=STRING              Output file (default: stdout)
      --user-agent="cewlai/1.0"    User agent for crawler
  -v, --verbose                    Verbose output
      --version                    Print version and exit
      --update                     Self-update to latest release
      --ai                         Enable AI enrichment
  -p, --provider=STRING            AI provider: anthropic, openai, groq,
                                   openrouter, cerebras, huggingface
  -m, --model=STRING               Model name or shorthand
      --api-key=STRING             API key (or use env vars)
      --base-url=STRING            Custom API base URL for OpenAI-compatible
                                   endpoints
      --list-models                List available models for the selected
                                   provider
      --mode="default"             AI prompt mode: default, passwords, dirs,
                                   subdomains, geo
      --prompt=STRING              Custom AI system prompt (overrides --mode)
      --ai-words=200               Number of AI-generated words
      --ai-context=4000            Max characters of context sent to LLM
  -e, --email                      Extract email addresses
      --email-file=STRING          Write emails to file
  -a, --meta                       Extract document metadata
      --meta-file=STRING           Write metadata to file
      --min-word-length=3          Minimum word length
      --max-word-length=0          Maximum word length (0 = no limit)
      --lowercase                  Lowercase all words
      --with-numbers               Include words with numbers
  -c, --count                      Show word frequency count
  -g, --groups=0                   Generate word groups of N
      --mutate                     Generate word mutations (leet, reverse,
                                   suffixes like CUPP)
      --mutate-config=STRING       Custom mutation config file (JSON)
      --offsite                    Follow offsite links
      --proxy=STRING               HTTP proxy URL
      --auth-type=STRING           Auth type: basic
      --auth-user=STRING           Auth username
      --auth-pass=STRING           Auth password
      --header=HEADER,...          Custom header (repeatable, Key: Value)
      --exclude=STRING             File with paths to exclude
      --max-pages=0                Maximum pages to crawl (0 = no limit)
  -t, --threads=2                  Number of concurrent crawl threads
      --capture-paths              Add URL path components to wordlist
      --capture-subdomains         Add subdomains to wordlist
      --capture-domain             Add domain to wordlist
```

## Security and Privacy

> [!CAUTION]
> Cloud AI providers (Groq, OpenRouter, Cerebras, HuggingFace, Anthropic, OpenAI) receive the crawled context from your target site when you use `--ai`. This includes text content, page titles, metadata, and any other data extracted during the crawl.
>
> You have no control over what these providers log, store, retain, or use for model training. Sending client data to a third-party API without authorization may violate your rules of engagement, NDA, or data protection regulations (GDPR, HIPAA, etc.).

> [!TIP]
> For sensitive engagements, use a local model to keep all data on your machine:
> ```bash
> ollama pull llama3
> cewlai -u https://example.com --ai -p openai -m llama3 \
>   --base-url http://localhost:11434/v1 --api-key dummy
> ```
> No external API calls. No data leaves your network.

---

## AI Providers

> [!NOTE]
> Tested with Groq and Cerebras. Other providers are supported but not yet fully tested. If you run into issues, please open an issue.

### Paid

| Provider  | Flag           | Models                                                                                         |
| --------- | -------------- | ---------------------------------------------------------------------------------------------- |
| Anthropic | `-p anthropic` | `haiku`, `sonnet`, `opus`                                                                      |
| OpenAI    | `-p openai`    | `gpt-4.1-mini`, `gpt-4.1`, `gpt-4.1-nano`, `gpt-4o-mini`, `gpt-4o`, `o3-mini`, `o3`, `o4-mini` |

### Free (no credit card)

| Provider    | Flag             | Default Model           | Env Var              |
| ----------- | ---------------- | ----------------------- | -------------------- |
| Groq        | `-p groq`        | llama-3.3-70b-versatile | `GROQ_API_KEY`       |
| OpenRouter  | `-p openrouter`  | openrouter/free         | `OPENROUTER_API_KEY` |
| Cerebras    | `-p cerebras`    | llama-3.3-70b           | `CEREBRAS_API_KEY`   |
| HuggingFace | `-p huggingface` | Llama-3.3-70B-Instruct  | `HF_TOKEN`           |

### Local (Ollama, LM Studio, vLLM)

```bash
cewlai -u https://example.com --ai -p openai -m llama3 --base-url http://localhost:11434/v1 --api-key dummy
```

### Proxy and Tor

The `--proxy` flag supports HTTP, HTTPS, and SOCKS5 proxies natively:

```bash
# HTTP proxy
cewlai -u https://example.com --proxy http://127.0.0.1:8080

# Tor (SOCKS5)
cewlai -u https://example.com --proxy socks5://127.0.0.1:9050
```

> [!TIP]
> cewlai is compiled with CGO enabled so proxychains should work too. However `--proxy` is more reliable and doesn't require external tools.

---

## AI Modes

| Mode         | Description                                                 |
| ------------ | ----------------------------------------------------------- |
| `default`    | General contextual words, industry terms, password patterns |
| `passwords`  | Likely passwords based on organization context              |
| `dirs`       | Hidden directories, endpoints, backup files                 |
| `subdomains` | Likely subdomains for the target                            |
| `geo`        | Geographic password patterns based on location              |

Custom prompt: `--prompt "Your custom system prompt here"`

## How AI Enrichment Works

1. The crawler visits pages and collects text per page from all sources (HTML, JS, XML, JSON, CSS, metadata, subtitles)
2. A context summary is built by sampling text evenly across all crawled pages (randomized order, default 4000 chars, configurable with `--ai-context`). This ensures the LLM sees the full breadth of the site, not just the first few pages
3. The context is sent to the LLM with a specialized prompt (comma-separated output to save tokens). The tool retries until the exact requested word count is reached (`--ai-words`), deduplicating across attempts
4. The LLM generates contextually related words that are NOT on the site: industry terms, likely passwords, role names, product names, date patterns, location words
5. Crawled results are cached locally (default 60min TTL, `--no-cache` to bypass). Running different AI modes on the same target reuses the cached crawl instantly
6. Both crawled and AI-generated words are merged, deduplicated, and sorted

## Features vs CeWL

| Feature                    | CeWL             | CeWL AI                                     |
| -------------------------- | ---------------- | ------------------------------------------- |
| Web crawling               | Yes              | Yes                                         |
| Word extraction            | Yes              | Yes (goquery, cleaner parsing)              |
| Email extraction           | Yes              | Yes (+ deobfuscation: `[at]`, `(at)`, etc.) |
| Document metadata          | Yes (exiftool)   | Yes (native Go, no external deps)           |
| URL component capture      | Yes              | Yes                                         |
| Word groups                | Yes              | Yes                                         |
| Word count                 | Yes              | Yes                                         |
| Proxy/Auth/Headers         | Yes              | Yes                                         |
| AI enrichment              | No               | Yes (6 providers + local)                   |
| AI prompt modes            | No               | Yes (5 modes + custom)                      |
| Single binary              | No (Ruby + gems) | Yes (Go)                                    |
| Self-update                | No               | Yes                                         |
| TLS skip                   | No               | Yes                                         |
| Obfuscated email detection | No               | Yes                                         |
| JavaScript parsing         | No               | Yes (jsluice, inline + external .js)        |
| XML/RSS/Atom/SVG parsing   | No               | Yes (sitemap, feeds, SVG text)              |
| JSON parsing               | No               | Yes (APIs, manifests, configs)              |
| CSS parsing                | No               | Yes (selectors, variables, URLs, comments)  |
| Audio/Video metadata       | No               | Yes (ID3, MP4, OGG - title, artist, album)  |
| Subtitle extraction        | No               | Yes (VTT + SRT transcript text)             |
| Word mutations (CUPP-like) | No               | Yes (leet, reverse, suffixes, custom JSON)  |
| AI word count control      | No               | Yes (`--ai-words` with retry loop)          |
| AI context control         | No               | Yes (`--ai-context` configurable)           |
| Token optimization         | No               | Yes (comma-separated output)                |
| Model listing              | No               | Yes (`--list-models` queries provider API)  |
| API key validation         | No               | Yes (tells you which env var to set)        |
| Tor/SOCKS5 proxy           | No               | Yes (`--proxy socks5://...`)                |
| Concurrent crawling        | No (sequential)  | Yes (`-t` configurable threads)             |
| FTP crawling               | No               | Yes (anonymous + auth, parallel downloads)  |
| SMB crawling               | No               | Yes (SMB2/3, NTLMv2 auth, URL credentials)  |
| Resource following         | `<a>` only       | `<a>` for navigation + separate collector for `<script>`, `<link>`, `<img>`, `<iframe>`, `<track>` (no depth cost) |
| Error page extraction      | No               | Yes (words from 404, 500, etc.)             |
| JS URL discovery           | No               | Yes (jsluice URLs are visited)              |
| HTML comment extraction    | No               | Yes                                         |

## Library Usage

The packages are importable for use in your own Go tools:

```go
package main

import (
	"context"
	"fmt"

	"github.com/Chocapikk/cewlai/ai"
	"github.com/Chocapikk/cewlai/crawler"
	"github.com/Chocapikk/cewlai/words"
)

func main() {
	// Crawl a target (HTTP, HTTPS, FTP, or SMB)
	ctx := context.Background()
	result, _ := crawler.Crawl(ctx, crawler.CrawlOptions{
		URL:           "https://example.com",
		Depth:         2,
		UserAgent:     "mybot/1.0",
		ExtractEmails: true,
		ExtractMeta:   true,
	})

	// Filter and deduplicate
	filtered := words.FilterWords(result.Words, 3, 0, true)
	filtered = words.DeduplicateWords(filtered)

	// AI enrichment
	provider, _ := ai.NewAIProvider("groq", "", "", "")
	prompt := ai.ResolvePrompt("passwords", "", 200)
	maxTokens := ai.MaxTokensForWords(200)
	aiWords, _ := provider.GenerateWords(ctx, result, prompt, maxTokens)

	// Merge everything
	final := words.DeduplicateWords(filtered, aiWords)
	for _, w := range final {
		fmt.Println(w)
	}
}
```

### Available packages

| Package          | Import                                        | Description                                                      |
| ---------------- | --------------------------------------------- | ---------------------------------------------------------------- |
| `crawler`        | `github.com/Chocapikk/cewlai/crawler`        | HTTP/FTP/SMB crawling, Source interface, cache, options           |
| `crawler/parser` | `github.com/Chocapikk/cewlai/crawler/parser`  | Content parsers (HTML, JS, XML, JSON, CSS, PDF, Office, media)   |
| `words`          | `github.com/Chocapikk/cewlai/words`           | Word splitting, filtering, dedup, counting, grouping, mutations  |
| `ai`             | `github.com/Chocapikk/cewlai/ai`              | LLM providers, prompt modes, response parsing, model listing     |

## Origin

CeWL has been the go-to wordlist generator since 2012, but it was built in a pre-AI era. It just looks for words on the webpage and that's it. The idea behind this project came from a simple observation: CeWL is kind of old and probably pre-AI, so someone could probably make a more accurate version that uses AI to generate "like words", "industry similar terms", and contextually related passwords. We figured it probably already existed. It didn't.

## Credits

Created by [@Chocapikk](https://github.com/Chocapikk). Original idea by [@stlthr4k3r](https://github.com/stlthr4k3r).

Inspired by [CeWL](https://github.com/digininja/CeWL) by Robin Wood.
