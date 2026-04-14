package parser

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"testing"
)

func randomHex(n int) string {
	b := make([]byte, n/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func TestNewSecretScanner(t *testing.T) {
	s := NewSecretScanner()
	if s == nil {
		t.Fatal("NewSecretScanner returned nil")
	}
	if len(s.detectors) == 0 {
		t.Fatal("NewSecretScanner has no detectors")
	}
}

func TestScanShortData(t *testing.T) {
	s := NewSecretScanner()
	got := s.Scan([]byte("abc"), "test.txt")
	if got != nil {
		t.Errorf("expected nil for short data, got %v", got)
	}
}

func TestScanNoSecrets(t *testing.T) {
	s := NewSecretScanner()
	got := s.Scan([]byte("This is a completely normal sentence with no secrets at all."), "readme.md")
	if len(got) != 0 {
		t.Errorf("expected no findings for benign text, got %d", len(got))
	}
}

func TestScanFindsSecrets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFind bool
	}{
		{
			name:     "Stripe Live Key",
			input:    "sk_live_51" + randomHex(40),
			wantFind: true,
		},
		{
			name:     "Slack Bot Token",
			input:    fmt.Sprintf("xoxb-%d-%d-%s", 41521398724+mrand.Int63()%1000, 8174928371924+mrand.Int63()%1000, randomHex(24)),
			wantFind: true,
		},
		{
			name:     "GitLab PAT",
			input:    "glpat-" + randomHex(20),
			wantFind: true,
		},
		{
			name:     "DigitalOcean PAT",
			input:    "dop_v1_" + randomHex(64),
			wantFind: true,
		},
		{
			name:     "Doppler Token",
			input:    "dp.pt." + randomHex(40),
			wantFind: true,
		},
		{
			name:     "SendGrid API Key",
			input:    fmt.Sprintf("SG.%s.%s", randomHex(22), randomHex(43)),
			wantFind: true,
		},
		{
			name:     "Regular text",
			input:    "Hello, this is just regular text without any secrets",
			wantFind: false,
		},
	}

	s := NewSecretScanner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := s.Scan([]byte(tt.input), "test-source")
			found := len(results) > 0
			if found != tt.wantFind {
				if tt.wantFind {
					t.Errorf("expected to find secret but didn't")
				} else {
					t.Errorf("did not expect to find secret but found %d", len(results))
				}
			}
			if found {
				for _, r := range results {
					if r.DetectorName == "" {
						t.Error("finding has empty DetectorName")
					}
					if r.Source != "test-source" {
						t.Errorf("expected source 'test-source', got %q", r.Source)
					}
				}
			}
		})
	}
}

func TestHasKeyword(t *testing.T) {
	tests := []struct {
		data     string
		keywords []string
		want     bool
	}{
		{"contains sk_live key", []string{"sk_live"}, true},
		{"CONTAINS SK_LIVE KEY", []string{"sk_live"}, true},
		{"no match here", []string{"xoxb", "glpat"}, false},
		{"anything matches", nil, true},
		{"anything matches", []string{}, true},
	}
	for _, tt := range tests {
		got := hasKeyword(tt.data, tt.keywords)
		if got != tt.want {
			t.Errorf("hasKeyword(%q, %v) = %v, want %v", tt.data, tt.keywords, got, tt.want)
		}
	}
}

func BenchmarkScan_NoSecrets(b *testing.B) {
	s := NewSecretScanner()
	data := []byte("This is a normal document with no secrets, just some regular text content about a company and their products")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		s.Scan(data, "bench.txt")
	}
}

func BenchmarkScan_WithSecret(b *testing.B) {
	s := NewSecretScanner()
	data := []byte("config: sk_live_51" + randomHex(40))
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		s.Scan(data, "bench.txt")
	}
}
