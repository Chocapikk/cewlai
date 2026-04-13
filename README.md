# CeWL AI

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

With a free provider:

```bash
cewlai -u https://example.com --ai -p groq
```

Full example:

```bash
cewlai -u https://target.com -d 3 --ai -p anthropic -m haiku \
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
      --mode="default"             AI prompt mode: default, passwords, dirs,
                                   subdomains, geo
      --prompt=STRING              Custom AI system prompt (overrides --mode)
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
      --offsite                    Follow offsite links
      --proxy=STRING               HTTP proxy URL
      --auth-type=STRING           Auth type: basic
      --auth-user=STRING           Auth username
      --auth-pass=STRING           Auth password
      --header=HEADER,...          Custom header (repeatable, Key: Value)
      --exclude=STRING             File with paths to exclude
      --max-pages=0                Maximum pages to crawl (0 = no limit)
      --capture-paths              Add URL path components to wordlist
      --capture-subdomains         Add subdomains to wordlist
      --capture-domain             Add domain to wordlist
```

## AI Providers

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
cewlai -u https://target.com --ai -p openai -m llama3 --base-url http://localhost:11434/v1 --api-key dummy
```

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

1. The crawler extracts visible text from the target site
2. A context summary (up to 4000 chars) is sent to the LLM
3. The LLM generates contextually related words that are NOT on the site: industry terms, likely passwords, role names, product names, date patterns, location words, common mutations
4. Both crawled and AI-generated words are merged, deduplicated, and sorted

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

## Credits

Created by [@Chocapikk](https://github.com/Chocapikk). Original idea by [@stlthr4k3r](https://github.com/stlthr4k3r).

Inspired by [CeWL](https://github.com/digininja/CeWL) by Robin Wood.
