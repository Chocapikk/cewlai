# AI Providers

CeWL AI supports multiple LLM providers out of the box. This document lists all supported providers, how to get an API key, and example usage.

> **Tested with Groq and Cerebras.** Other providers are supported but not yet fully tested. If you run into issues, please open an issue.

> **Tip:** Use `cewlai --list-models -p <provider>` to see available models for any provider.

> **Privacy warning:** Cloud providers receive the crawled context from your target site. During a real pentest engagement, you don't control what these providers log, store, or use for training. If the target data is sensitive, use a local model via `--base-url` (Ollama, LM Studio, vLLM) to keep everything on your machine.

## Anthropic

- **Website**: https://console.anthropic.com/
- **Pricing**: Paid (usage-based)
- **Env var**: `ANTHROPIC_API_KEY`
- **Models**: `haiku` (fast, cheap), `sonnet` (balanced), `opus` (most capable)

```bash
export ANTHROPIC_API_KEY=sk-ant-...
cewlai -u https://example.com --ai -p anthropic -m sonnet
```

## OpenAI

- **Website**: https://platform.openai.com/api-keys
- **Pricing**: Paid (usage-based)
- **Env var**: `OPENAI_API_KEY`
- **Models**: `gpt-4.1-mini` (cheap), `gpt-4.1` (balanced), `gpt-4.1-nano` (cheapest), `gpt-4o-mini`, `gpt-4o`, `o3-mini`, `o3`, `o4-mini`

```bash
export OPENAI_API_KEY=sk-...
cewlai -u https://example.com --ai -p openai -m gpt-4.1-mini
```

## Groq (free)

- **Website**: https://console.groq.com/keys
- **Pricing**: Free, no credit card required
- **Rate limits**: ~30 req/min, 1000 req/day
- **Env var**: `GROQ_API_KEY`
- **Default model**: llama-3.3-70b-versatile

```bash
export GROQ_API_KEY=gsk_...
cewlai -u https://example.com --ai -p groq
```

## OpenRouter (free tier)

- **Website**: https://openrouter.ai/keys
- **Pricing**: Free models available, no credit card required
- **Rate limits**: 20 req/min, 200 req/day on free models
- **Env var**: `OPENROUTER_API_KEY`
- **Default model**: openrouter/free (auto-selects best available free model)

```bash
export OPENROUTER_API_KEY=sk-or-...
cewlai -u https://example.com --ai -p openrouter
```

## Cerebras (free)

- **Website**: https://cloud.cerebras.ai/
- **Pricing**: Free, no credit card required
- **Rate limits**: 30 req/min, 1M tokens/day
- **Env var**: `CEREBRAS_API_KEY`
- **Default model**: llama-3.3-70b

```bash
export CEREBRAS_API_KEY=csk-...
cewlai -u https://example.com --ai -p cerebras
```

## HuggingFace (free)

- **Website**: https://huggingface.co/settings/tokens
- **Pricing**: Free tier available
- **Env var**: `HF_TOKEN`
- **Default model**: meta-llama/Llama-3.3-70B-Instruct

```bash
export HF_TOKEN=hf_...
cewlai -u https://example.com --ai -p huggingface
```

## opencode (local server)

Instead of calling a cloud provider directly, you can route AI enrichment through a **local `opencode serve`** instance. opencode is not OpenAI-compatible, so this provider talks to opencode's own HTTP API and reuses whatever models/providers you already configured in opencode (e.g. `opencode/big-pickle`, `bankofai/glm-5.3-flash`).

- **Start opencode**: `opencode serve` (default `http://localhost:4096`)
- **Env var**: none for an unsecured server. If you started `opencode serve` with `OPENCODE_SERVER_PASSWORD`, give cewlai the same value through `--api-key` or export `OPENCODE_SERVER_PASSWORD` for it too. It is sent as HTTP Basic credentials, which is what the server expects.
- **Model format**: `-m providerID/modelID` (slash is optional; defaults to `opencode/`)
- **Default model**: `big-pickle`

```bash
opencode serve
# in another terminal:
cewlai -u https://example.com --ai -p opencode -m big-pickle
# or pick a specific configured provider/model:
cewlai -u https://example.com --ai -p opencode -m bankofai/glm-5.3-flash
# non-default port:
cewlai -u https://example.com --ai -p opencode --base-url http://localhost:5000 -m big-pickle
# list the models opencode exposes to you:
cewlai --list-models -p opencode
# password-protected server:
OPENCODE_SERVER_PASSWORD=secret opencode serve
cewlai -u https://example.com --ai -p opencode --api-key secret
```

> **Note:** opencode runs a full agent loop, so each batch call is slower than a direct model request. The `--ai-words` target and retry loop in `enrichWithAI` still apply.

> **Note:** `--ai-words` still drives the retry loop here, but the per-request token ceiling it derives is not applied. opencode's message API exposes no token limit, so the cap is whatever the server side model enforces.

> **Privacy:** Because the crawl context never leaves your machine (it only goes to your local opencode server), this is the recommended option for sensitive engagements.

## Local models (Ollama, LM Studio, vLLM)

Any OpenAI-compatible endpoint works via `--base-url`. No API key needed for local models (pass `dummy` as key).

### Ollama

- **Website**: https://ollama.com/
- **Install**: `curl -fsSL https://ollama.com/install.sh | sh`
- **Pull a model**: `ollama pull llama3`

```bash
cewlai -u https://example.com --ai -p openai -m llama3 --base-url http://localhost:11434/v1 --api-key dummy
```

### LM Studio

- **Website**: https://lmstudio.ai/
- Start the local server, then:

```bash
cewlai -u https://example.com --ai -p openai -m your-model --base-url http://localhost:1234/v1 --api-key dummy
```

### vLLM

- **Website**: https://github.com/vllm-project/vllm

```bash
cewlai -u https://example.com --ai -p openai -m your-model --base-url http://localhost:8000/v1 --api-key dummy
```

## Custom endpoints

Any service exposing an OpenAI-compatible `/v1/chat/completions` endpoint works:

```bash
cewlai -u https://example.com --ai -p openai -m model-name --base-url https://your-endpoint.com/v1 --api-key your-key
```
