package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Chocapikk/cewlai/crawler"
)

const outputRule = `

IMPORTANT: Every word you generate MUST be directly derived from the context provided. Do not generate random or generic words. Base everything on what the site actually contains.

Output ONLY words separated by commas on a single line. No explanations, no numbering, no markdown, no sentences. Example: word1,word2,word3`

var PromptModes = map[string]string{
	"default": `You are a security-focused wordlist generator for penetration testing.
Given context from a crawled website, generate words that are contextually related but DO NOT appear on the website itself.

Think about:
- Industry-specific terminology and jargon
- Common password patterns related to the organization (company name + year, product names, etc.)
- Employee roles and department names that likely exist
- Product names, codenames, and abbreviations
- Related technical terms and acronyms
- Date-based patterns (founding year, current year, seasons)
- Location-based words (city, country, region)
- Common mutations (capitalize, leet speak, append numbers/symbols)` + outputRule,

	"passwords": `You are a password wordlist generator for penetration testing.
Given context from a crawled website, generate likely passwords that employees or users of this organization might use.

Think about:
- Company name with years, seasons, symbols (Acme2024!, acme_summer)
- City or country with numbers (Paris75, london2024)
- Product names and project codenames as passwords
- Department names (admin, finance, hr, marketing, devops)
- Common patterns: Name+Year, Name+123, Name+!, Welcome1, P@ssw0rd
- Keyboard patterns mixed with context (Qwerty+company)
- Default and lazy passwords related to the industry` + outputRule,

	"dirs": `You are a directory and path wordlist generator for penetration testing.
Given context from a crawled website, generate likely directory names, file paths, and endpoints that might exist on this web application but are not linked.

Think about:
- Admin panels (/admin, /dashboard, /panel, /manage)
- API endpoints (/api/v1, /api/v2, /graphql)
- Backup files (.bak, .old, .backup, .sql, .dump)
- Config files (.env, .config, web.config, .htaccess)
- Common CMS paths based on detected technology
- Development/staging paths (/dev, /staging, /test, /debug)
- Documentation paths (/docs, /swagger, /openapi)
- Industry-specific paths based on the site context` + outputRule,

	"subdomains": `You are a subdomain wordlist generator for penetration testing.
Given context from a crawled website, generate likely subdomains that this organization might use.

Think about:
- Standard infrastructure (mail, vpn, ftp, ssh, ns1, ns2, dns)
- Development (dev, staging, test, qa, uat, sandbox, beta)
- Internal tools (jira, confluence, gitlab, jenkins, grafana, kibana)
- Cloud and CDN (cdn, assets, static, media, storage, s3)
- Industry-specific services based on the organization type
- Product names and project codenames as subdomains
- Regional subdomains (us, eu, asia, fr, uk)
- API subdomains (api, api-v2, gateway, ws)` + outputRule,

	"geo": `You are a geographic password wordlist generator for penetration testing.
Given context from a crawled website, generate passwords based on geographic locations related to this organization.

Think about:
- City names with numbers and symbols (Paris2024!, lyon01)
- Postal codes and area codes
- Nearby cities and regions
- Country-specific patterns
- Street names and landmarks near the organization
- Regional slang and abbreviations
- Seasonal and local event names
- Sports team names from the area` + outputRule,
}

var SystemPrompt = PromptModes["default"]

func ResolvePrompt(mode, customPrompt string, maxWords int) string {
	base := PromptModes["default"]
	if customPrompt != "" {
		base = customPrompt + outputRule
	} else if p, ok := PromptModes[mode]; ok {
		base = p
	}
	if maxWords > 0 {
		base += fmt.Sprintf("\n\nGenerate exactly %d words, no more, no less.", maxWords)
	}
	return base
}

type AIProvider interface {
	GenerateWords(ctx context.Context, result *crawler.CrawlResult, prompt string, maxTokens int) ([]string, error)
}

var providerPresets = map[string]struct {
	baseURL      string
	defaultModel string
	envKey       string
}{
	"groq":        {"https://api.groq.com/openai/v1", "llama-3.3-70b-versatile", "GROQ_API_KEY"},
	"openrouter":  {"https://openrouter.ai/api/v1", "openrouter/free", "OPENROUTER_API_KEY"},
	"cerebras":    {"https://api.cerebras.ai/v1", "llama-3.3-70b", "CEREBRAS_API_KEY"},
	"huggingface": {"https://router.huggingface.co/v1", "meta-llama/Llama-3.3-70B-Instruct", "HF_TOKEN"},
}

func NewAIProvider(provider, apiKey, model, baseURL string) (AIProvider, error) {
	p := strings.ToLower(provider)

	if preset, ok := providerPresets[p]; ok {
		if baseURL == "" {
			baseURL = preset.baseURL
		}
		if model == "" {
			model = preset.defaultModel
		}
		if apiKey == "" {
			apiKey = os.Getenv(preset.envKey)
		}
		return newOpenAIProvider(apiKey, model, baseURL), nil
	}

	switch p {
	case "anthropic":
		return newAnthropicProvider(apiKey, model, baseURL), nil
	case "openai":
		return newOpenAIProvider(apiKey, model, baseURL), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: anthropic, openai, groq, openrouter, cerebras, huggingface)", provider)
	}
}

func MaxTokensForWords(n int) int {
	tokens := n * 3
	if tokens < 4096 {
		return 4096
	}
	if tokens > 16384 {
		return 16384
	}
	return tokens
}

func BuildUserMessage(result *crawler.CrawlResult) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Target URL: %s\n", result.URL)
	if result.Title != "" {
		fmt.Fprintf(&sb, "Page Title: %s\n", result.Title)
	}
	fmt.Fprintf(&sb, "Pages crawled: %d\n\nExtracted context:\n%s", result.Pages, result.Context)
	return sb.String()
}

func ParseAIResponse(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		for _, w := range strings.Split(line, ",") {
			w = strings.TrimSpace(w)
			w = strings.TrimLeft(w, "-*# ")
			if w == "" || strings.Contains(w, " ") {
				continue
			}
			out = append(out, w)
		}
	}
	return out
}
