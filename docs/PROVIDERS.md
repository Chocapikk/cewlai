# AI Providers

CeWL AI supports multiple LLM providers out of the box. This document lists all supported providers, how to get an API key, and example usage.

> **Tested with Groq and Cerebras.** Other providers are supported but not yet fully tested. If you run into issues, please open an issue.

> **Tip:** Use `cewlai --list-models -p <provider>` to see available models for any provider.

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
