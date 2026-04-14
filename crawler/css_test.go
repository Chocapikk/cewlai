package crawler

import "testing"

func TestExtractFromCSS_Selectors(t *testing.T) {
	css := []byte(`
		.admin-panel { color: red; }
		#login-form { display: none; }
		.user-profile { margin: 10px; }
		.btn-primary { background: blue; }
	`)

	wordSet := make(map[string]struct{})
	extractFromCSS(css, wordSet)

	expected := []string{"admin-panel", "login-form", "user-profile", "btn-primary"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q in wordSet, not found", w)
		}
	}
}

func TestExtractFromCSS_Variables(t *testing.T) {
	css := []byte(`
		:root {
			--brand-color: #ff0000;
			--company-blue: #0000ff;
			--font-primary: 'Arial';
		}
	`)

	wordSet := make(map[string]struct{})
	extractFromCSS(css, wordSet)

	expected := []string{"brand-color", "company-blue", "font-primary"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q in wordSet, not found", w)
		}
	}
}

func TestExtractFromCSS_URLs(t *testing.T) {
	css := []byte(`
		.logo { background: url('/images/company-logo.png'); }
		.banner { background: url("https://cdn.example.com/assets/header.jpg"); }
	`)

	wordSet := make(map[string]struct{})
	extractFromCSS(css, wordSet)

	if _, ok := wordSet["company-logo"]; !ok {
		t.Error("expected 'company-logo' from URL path")
	}
}

func TestExtractFromCSS_Comments(t *testing.T) {
	css := []byte(`
		/* Admin dashboard styles */
		.dashboard { color: black; }
		/* Created by design team for internal portal */
	`)

	wordSet := make(map[string]struct{})
	extractFromCSS(css, wordSet)

	expected := []string{"Admin", "dashboard", "internal", "portal"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q from CSS comment, not found", w)
		}
	}
}

func TestExtractFromCSS_Empty(t *testing.T) {
	wordSet := make(map[string]struct{})
	extractFromCSS([]byte(""), wordSet)
	if len(wordSet) != 0 {
		t.Errorf("expected empty wordSet, got %d", len(wordSet))
	}
}

func BenchmarkExtractFromCSS(b *testing.B) {
	css := []byte(`
		.sparkle-container { display: flex; }
		#depth-section { visibility: hidden; }
		--presence-color: #gold;
		/* She wants her intensity to be chosen */
	`)

	for b.Loop() {
		wordSet := make(map[string]struct{})
		extractFromCSS(css, wordSet)
	}
}
