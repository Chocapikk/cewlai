package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Chocapikk/cewlai/crawler"
)

// DefaultOpenCodeBaseURL is used when --base-url is not provided and the
// provider is "opencode". Matches the opencode serve default port.
const DefaultOpenCodeBaseURL = "http://localhost:4096"

// Expected timeout for a single opencode message round-trip. opencode runs a
// full agent loop, which is slower than a direct model call.
const openCodeRequestTimeout = 300 * time.Second

// OpenCodeServerPasswordEnv is the environment variable opencode itself uses
// to protect `opencode serve`. When the server is started with it, every
// request must carry HTTP Basic credentials with the literal user "opencode"
// and that value as the password.
const OpenCodeServerPasswordEnv = "OPENCODE_SERVER_PASSWORD"

// openCodeBasicAuthUser is the only username a password-protected opencode
// server accepts. Anything else is rejected with 401.
const openCodeBasicAuthUser = "opencode"

// opencodeProvider talks to a local opencode server's own HTTP API
// (POST /session, POST /session/{id}/message). It is not OpenAI-compatible,
// so it does not use the openai SDK.
type opencodeProvider struct {
	baseURL    string
	modelID    string
	providerID string
	password   string
	client     *http.Client
}

func newOpencodeProvider(apiKey, model, baseURL string) *opencodeProvider {
	if baseURL == "" {
		baseURL = DefaultOpenCodeBaseURL
	}
	providerID, modelID := parseOpencodeModel(model)
	return &opencodeProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		modelID:    modelID,
		providerID: providerID,
		password:   opencodeServerPassword(apiKey),
		client: &http.Client{
			Timeout: openCodeRequestTimeout,
		},
	}
}

// opencodeServerPassword resolves the server password from --api-key first,
// then from the environment. An unsecured server needs neither.
func opencodeServerPassword(apiKey string) string {
	if apiKey != "" {
		return apiKey
	}
	return os.Getenv(OpenCodeServerPasswordEnv)
}

// setOpenCodeAuth adds Basic credentials when a password is configured. Sending
// nothing against an unsecured server is fine, so this is a no-op then.
func setOpenCodeAuth(req *http.Request, password string) {
	if password != "" {
		req.SetBasicAuth(openCodeBasicAuthUser, password)
	}
}

// parseOpencodeModel splits a "providerID/modelID" spec into its parts. If no
// slash is present, the whole value is a modelID and providerID defaults to
// "opencode". An empty input yields the opencode default model.
func parseOpencodeModel(model string) (providerID, modelID string) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "opencode", "big-pickle"
	}
	if i := strings.IndexByte(model, '/'); i > 0 && i < len(model)-1 {
		return model[:i], model[i+1:]
	}
	// A leading or trailing slash would yield an empty providerID or modelID,
	// which the server rejects, so treat the value as a bare modelID.
	return "opencode", strings.Trim(model, "/")
}

// GenerateWords creates a fresh session, sends the user context as a prompt to
// the configured opencode model, and returns the comma/line separated words
// found in the text parts of the reply.
//
// maxTokens is accepted to satisfy AIProvider but has no effect here: the
// opencode message API (POST /session/{id}/message) exposes no token limit,
// the underlying model is chosen server side. Callers relying on a hard cap
// should use a provider that supports one.
func (p *opencodeProvider) GenerateWords(ctx context.Context, result *crawler.CrawlResult, prompt string, _ int) ([]string, error) {
	sessionID, err := p.createSession(ctx)
	if err != nil {
		return nil, err
	}

	reply, err := p.sendMessage(ctx, sessionID, prompt, BuildUserMessage(result))
	if err != nil {
		return nil, err
	}

	return ParseAIResponse(reply), nil
}

func (p *opencodeProvider) createSession(ctx context.Context) (string, error) {
	body, err := json.Marshal(map[string]string{"title": "cewlai"})
	if err != nil {
		return "", fmt.Errorf("opencode: marshal session: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/session", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("opencode: build session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setOpenCodeAuth(req, p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("opencode: create session: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", p.errorFromResponse("create session", resp)
	}

	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return "", fmt.Errorf("opencode: parse session response: %w", err)
	}
	if out.ID == "" {
		return "", fmt.Errorf("opencode: session response missing id")
	}
	return out.ID, nil
}

type openCodeMessageRequest struct {
	Model  *openCodeModel `json:"model,omitempty"`
	System string         `json:"system,omitempty"`
	Parts  []openCodePart `json:"parts"`
}

type openCodeModel struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

type openCodePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openCodeMessageResponse struct {
	Parts []openCodePart `json:"parts"`
}

func (p *opencodeProvider) sendMessage(ctx context.Context, sessionID, prompt, userMessage string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("opencode: empty session id")
	}
	reqBody := openCodeMessageRequest{
		Model: &openCodeModel{
			ProviderID: p.providerID,
			ModelID:    p.modelID,
		},
		System: prompt,
		Parts: []openCodePart{
			{Type: "text", Text: userMessage},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("opencode: marshal message: %w", err)
	}

	url := fmt.Sprintf("%s/session/%s/message", p.baseURL, sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("opencode: build message request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setOpenCodeAuth(req, p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("opencode: send message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", p.errorFromResponse("send message", resp)
	}

	var out openCodeMessageResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&out); err != nil {
		return "", fmt.Errorf("opencode: parse message response: %w", err)
	}

	var sb strings.Builder
	for _, part := range out.Parts {
		if part.Type == "text" && part.Text != "" {
			sb.WriteString(part.Text)
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

func (p *opencodeProvider) errorFromResponse(op string, resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	msg := strings.TrimSpace(string(raw))
	if msg == "" {
		msg = resp.Status
	}
	if len(msg) > 1000 {
		msg = msg[:1000]
	}
	return fmt.Errorf("opencode: %s failed (status %d): %s", op, resp.StatusCode, msg)
}
