package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Chocapikk/cewlai/crawler"
)

// fakeOpenCodeServer is a minimal in-memory opencode server used to exercise
// the opencodeProvider against the same wire protocol opencode exposes.
type fakeOpenCodeServer struct {
	t       *testing.T
	mu      sync.Mutex
	message string
	parts   []map[string]interface{}
	// optional hook to inspect the received request body.
	onMessage func(body map[string]interface{})

	// negative-path controls
	sessionStatus int    // http status for POST /session (0 = 200)
	sessionBody   string // raw body for POST /session ("" = default)
	msgStatus     int    // http status for /message (0 = 200)
	msgBody       string // raw body for /message ("" = default)
}

func newFakeOpenCodeServer(t *testing.T, message string) *httptest.Server {
	f := &fakeOpenCodeServer{t: t, message: message}
	return f.serve()
}

// serve starts the fake server and registers its cleanup.
func (f *fakeOpenCodeServer) serve() *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(f.handle))
	if f.t != nil {
		f.t.Cleanup(srv.Close)
	}
	return srv
}

func (f *fakeOpenCodeServer) handle(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/session":
		if f.sessionStatus != 0 {
			http.Error(w, "session boom", f.sessionStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if f.sessionBody != "" {
			_, _ = w.Write([]byte(f.sessionBody))
			return
		}
		_, _ = w.Write([]byte(`{"id":"ses_test123"}`))

	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/session/") &&
		strings.HasSuffix(r.URL.Path, "/message"):
		if f.onMessage != nil {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.onMessage(body)
		}
		if f.msgStatus != 0 {
			http.Error(w, "message boom", f.msgStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if f.msgBody != "" {
			_, _ = w.Write([]byte(f.msgBody))
			return
		}
		resp := map[string]interface{}{
			"info": map[string]interface{}{"id": "msg_1"},
			"parts": []map[string]interface{}{
				{"type": "reasoning", "text": "thinking"},
				{"type": "text", "text": f.message},
				{"type": "step-finish", "text": ""},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)

	default:
		http.NotFound(w, r)
	}
}

func openCodeTestResult() *crawler.CrawlResult {
	return &crawler.CrawlResult{
		URL:     "https://example.com",
		Title:   "Example",
		Pages:   2,
		Context: "acme security training",
	}
}

func TestOpencodeProvider_GenerateWords(t *testing.T) {
	// Comma separated single-line output (outputRule compliant).
	srv := newFakeOpenCodeServer(t, "acme,training,security,certified")
	p := newOpencodeProvider("", "big-pickle", srv.URL)

	words, err := p.GenerateWords(context.Background(), openCodeTestResult(), "system prompt", 100)
	if err != nil {
		t.Fatalf("GenerateWords error: %v", err)
	}
	want := []string{"acme", "training", "security", "certified"}
	if !sliceEqual(words, want) {
		t.Errorf("GenerateWords = %v, want %v", words, want)
	}
}

func TestOpencodeProvider_IgnoresNonTextParts(t *testing.T) {
	// reasoning parts must not leak into the word list.
	srv := newFakeOpenCodeServer(t, "word1,word2")
	p := newOpencodeProvider("", "big-pickle", srv.URL)

	words, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100)
	if err != nil {
		t.Fatalf("GenerateWords error: %v", err)
	}
	for _, w := range words {
		if strings.Contains(w, "thinking") {
			t.Errorf("reasoning text leaked into words: %q", w)
		}
	}
}

func TestOpencodeProvider_SendsModelAndSystem(t *testing.T) {
	var got map[string]interface{}
	f := &fakeOpenCodeServer{t: t, message: "a,b"}
	f.onMessage = func(body map[string]interface{}) {
		got = body
	}
	capSrv := httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(capSrv.Close)

	capP := newOpencodeProvider("", "bankofai/glm-5.3-flash", capSrv.URL)
	if _, err := capP.GenerateWords(context.Background(), openCodeTestResult(), "my system", 100); err != nil {
		t.Fatalf("GenerateWords error: %v", err)
	}

	if got == nil {
		t.Fatal("message request body was not captured")
	}
	model := got["model"].(map[string]interface{})
	if model["providerID"] != "bankofai" {
		t.Errorf("providerID = %v, want bankofai", model["providerID"])
	}
	if model["modelID"] != "glm-5.3-flash" {
		t.Errorf("modelID = %v, want glm-5.3-flash", model["modelID"])
	}
	if got["system"] != "my system" {
		t.Errorf("system = %v, want 'my system'", got["system"])
	}
}

func TestOpencodeProvider_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	p := newOpencodeProvider("", "big-pickle", srv.URL)
	if _, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100); err == nil {
		t.Error("expected error when server returns 500")
	}
}

func TestOpencodeProvider_ConnectionError(t *testing.T) {
	// closed server -> dial error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	p := newOpencodeProvider("", "big-pickle", url)
	if _, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100); err == nil {
		t.Error("expected error when server is unreachable")
	}
}

// BenchmarkOpencodeProvider_GenerateWords measures the wire cost of the two
// HTTP round-trips (create session + send message) against a local fake
// server, i.e. the per-call overhead on top of the model's own latency.
func BenchmarkOpencodeProvider_GenerateWords(b *testing.B) {
	srv := newFakeOpenCodeServer(&testing.T{}, "acme,training,security")
	p := newOpencodeProvider("", "big-pickle", srv.URL)
	res := openCodeTestResult()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.GenerateWords(context.Background(), res, "prompt", 100); err != nil {
			b.Fatalf("GenerateWords error: %v", err)
		}
	}
}

// ---- Negative tests ----

func TestOpencodeProvider_SessionResponseMissingID(t *testing.T) {
	f := &fakeOpenCodeServer{t: t}
	f.sessionBody = `{"ok": true}` // no "id"
	srv := f.serve()

	p := newOpencodeProvider("", "big-pickle", srv.URL)
	if _, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100); err == nil {
		t.Error("expected error when session response has no id")
	}
}

func TestOpencodeProvider_SessionMalformedJSON(t *testing.T) {
	f := &fakeOpenCodeServer{t: t}
	f.sessionBody = `{not valid json`
	srv := f.serve()

	p := newOpencodeProvider("", "big-pickle", srv.URL)
	_, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100)
	if err == nil {
		t.Fatal("expected error on malformed session JSON")
	}
	if !strings.Contains(errString(err), "parse session") {
		t.Errorf("expected 'parse session' in error, got %q", errString(err))
	}
}

func TestOpencodeProvider_MessageMalformedJSON(t *testing.T) {
	f := &fakeOpenCodeServer{t: t}
	f.msgBody = `<html>SPA fallback -- not JSON</html>`
	srv := f.serve()

	p := newOpencodeProvider("", "big-pickle", srv.URL)
	if _, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100); err == nil {
		t.Error("expected error on malformed message JSON")
	}
}

func TestOpencodeProvider_ReasoningOnlyReturnsEmpty(t *testing.T) {
	// No text parts at all -> empty result, no error.
	f := &fakeOpenCodeServer{t: t}
	f.msgBody = `{"info":{"id":"m1"},"parts":[{"type":"reasoning","text":"thinking"}]}`
	srv := f.serve()

	p := newOpencodeProvider("", "big-pickle", srv.URL)
	words, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(words) != 0 {
		t.Errorf("expected 0 words for reasoning-only reply, got %v", words)
	}
}

func TestOpencodeProvider_NoPartsReturnsEmpty(t *testing.T) {
	f := &fakeOpenCodeServer{t: t}
	f.msgBody = `{"info":{"id":"m1"},"parts":[]}`
	srv := f.serve()

	p := newOpencodeProvider("", "big-pickle", srv.URL)
	words, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(words) != 0 {
		t.Errorf("expected 0 words for empty parts, got %v", words)
	}
}

func TestOpencodeProvider_MessageServerError(t *testing.T) {
	f := &fakeOpenCodeServer{t: t}
	f.msgStatus = http.StatusUnauthorized // 401
	srv := f.serve()

	p := newOpencodeProvider("", "big-pickle", srv.URL)
	if _, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100); err == nil {
		t.Error("expected error when message endpoint returns 401")
	}
}

func TestOpencodeProvider_SessionFailureSkipsMessage(t *testing.T) {
	// If the session request fails, the message endpoint must never be hit.
	messageHit := false
	f := &fakeOpenCodeServer{t: t}
	f.sessionStatus = http.StatusInternalServerError
	f.onMessage = func(body map[string]interface{}) {
		messageHit = true
	}
	srv := f.serve()

	p := newOpencodeProvider("", "big-pickle", srv.URL)
	if _, err := p.GenerateWords(context.Background(), openCodeTestResult(), "p", 100); err == nil {
		t.Error("expected error when session endpoint fails")
	}
	if messageHit {
		t.Error("message endpoint was hit even though session creation failed")
	}
}

func TestOpencodeProvider_EmptySessionID(t *testing.T) {
	// Direct call with empty session id -> error, no panic, no bad URL.
	srv := newFakeOpenCodeServer(t, "a,b")
	p := newOpencodeProvider("", "big-pickle", srv.URL)

	if _, err := p.sendMessage(context.Background(), "", "p", "u"); err == nil {
		t.Error("expected error for empty session id")
	}
}

func TestOpencodeProvider_ContextCancelled(t *testing.T) {
	f := &fakeOpenCodeServer{t: t}
	srv := f.serve()
	p := newOpencodeProvider("", "big-pickle", srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before request

	if _, err := p.GenerateWords(ctx, openCodeTestResult(), "p", 100); err == nil {
		t.Error("expected error when context is already cancelled")
	}
}

// errString returns the error's message or "" when nil, for compact asserts.
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// A password-protected `opencode serve` answers 401 unless the request carries
// HTTP Basic credentials with the literal user "opencode". These cover that.

func TestOpencodeProvider_SendsBasicAuthWhenPasswordSet(t *testing.T) {
	var gotUser, gotPass string
	var gotOK bool
	f := &fakeOpenCodeServer{t: t, message: "alpha, beta"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, gotOK = r.BasicAuth()
		f.handle(w, r)
	}))
	t.Cleanup(srv.Close)

	p := newOpencodeProvider("hunter2", "", srv.URL)
	if _, err := p.GenerateWords(context.Background(), &crawler.CrawlResult{}, "sys", 0); err != nil {
		t.Fatalf("GenerateWords: %v", err)
	}
	if !gotOK {
		t.Fatal("no Basic credentials sent")
	}
	if gotUser != openCodeBasicAuthUser {
		t.Errorf("user = %q, want %q", gotUser, openCodeBasicAuthUser)
	}
	if gotPass != "hunter2" {
		t.Errorf("password = %q, want %q", gotPass, "hunter2")
	}
}

func TestOpencodeProvider_NoAuthWhenNoPassword(t *testing.T) {
	t.Setenv(OpenCodeServerPasswordEnv, "")
	var sawAuth bool
	f := &fakeOpenCodeServer{t: t, message: "alpha"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			sawAuth = true
		}
		f.handle(w, r)
	}))
	t.Cleanup(srv.Close)

	p := newOpencodeProvider("", "", srv.URL)
	if _, err := p.GenerateWords(context.Background(), &crawler.CrawlResult{}, "sys", 0); err != nil {
		t.Fatalf("GenerateWords: %v", err)
	}
	if sawAuth {
		t.Error("Authorization header sent against an unsecured server")
	}
}

func TestOpencodeProvider_PasswordFromEnv(t *testing.T) {
	t.Setenv(OpenCodeServerPasswordEnv, "fromenv")
	p := newOpencodeProvider("", "", "http://example.invalid")
	if p.password != "fromenv" {
		t.Errorf("password = %q, want %q", p.password, "fromenv")
	}
}

func TestOpencodeProvider_APIKeyBeatsEnv(t *testing.T) {
	t.Setenv(OpenCodeServerPasswordEnv, "fromenv")
	p := newOpencodeProvider("fromflag", "", "http://example.invalid")
	if p.password != "fromflag" {
		t.Errorf("password = %q, want %q", p.password, "fromflag")
	}
}

func TestParseOpencodeModel_EdgeSlashes(t *testing.T) {
	cases := []struct {
		in        string
		wantProv  string
		wantModel string
	}{
		{"anthropic/claude", "anthropic", "claude"},
		{"big-pickle", "opencode", "big-pickle"},
		{"", "opencode", "big-pickle"},
		{"/big-pickle", "opencode", "big-pickle"},
		{"big-pickle/", "opencode", "big-pickle"},
	}
	for _, c := range cases {
		prov, model := parseOpencodeModel(c.in)
		if prov != c.wantProv || model != c.wantModel {
			t.Errorf("parseOpencodeModel(%q) = %q, %q, want %q, %q", c.in, prov, model, c.wantProv, c.wantModel)
		}
		if prov == "" || model == "" {
			t.Errorf("parseOpencodeModel(%q) produced an empty part, the server rejects those", c.in)
		}
	}
}
