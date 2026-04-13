package ai

import (
	"testing"

	"github.com/Chocapikk/cewlai/crawler"
)

func TestBuildUserMessage(t *testing.T) {
	result := &crawler.CrawlResult{
		URL:     "https://example.com",
		Title:   "Example",
		Pages:   3,
		Context: "some context",
	}
	msg := BuildUserMessage(result)
	if msg == "" {
		t.Error("BuildUserMessage returned empty string")
	}
	for _, expected := range []string{"https://example.com", "Example", "3", "some context"} {
		if !contains(msg, expected) {
			t.Errorf("BuildUserMessage missing %q in output", expected)
		}
	}
}

func TestParseAIResponse(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"word1\nword2\nword3", []string{"word1", "word2", "word3"}},
		{"- word1\n* word2\n# word3", []string{"word1", "word2", "word3"}},
		{"this is a sentence\nword", []string{"word"}},
		{"  \n\n  \n", nil},
		{"  spaced  \n\ttabbed\t", []string{"spaced", "tabbed"}},
	}
	for _, tt := range tests {
		got := ParseAIResponse(tt.input)
		if !sliceEqual(got, tt.want) {
			t.Errorf("ParseAIResponse(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNewAIProvider_UnknownProvider(t *testing.T) {
	_, err := NewAIProvider("invalid", "", "", "")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNewAIProvider_Presets(t *testing.T) {
	for _, name := range []string{"groq", "openrouter", "cerebras", "huggingface"} {
		p, err := NewAIProvider(name, "dummy-key", "", "")
		if err != nil {
			t.Errorf("NewAIProvider(%q) error: %v", name, err)
		}
		if p == nil {
			t.Errorf("NewAIProvider(%q) returned nil", name)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func sliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
