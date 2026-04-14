package parser

import "testing"

func TestExtractFromJSON_Object(t *testing.T) {
	data := []byte(`{
		"name": "Chocapikk",
		"role": "Security Researcher",
		"tools": ["metasploit", "burpsuite", "nmap"],
		"nested": {
			"company": "VulnCheck",
			"location": "Paris"
		}
	}`)

	wordSet := make(map[string]struct{})
	ExtractFromJSON(data, wordSet)

	expected := []string{"Chocapikk", "Security", "Researcher", "metasploit", "burpsuite", "nmap", "VulnCheck", "Paris", "name", "role", "tools", "company", "location"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q in wordSet, not found", w)
		}
	}
}

func TestExtractFromJSON_Array(t *testing.T) {
	data := []byte(`[{"title": "Exploit Development"}, {"title": "Reverse Engineering"}]`)

	wordSet := make(map[string]struct{})
	ExtractFromJSON(data, wordSet)

	expected := []string{"Exploit", "Development", "Reverse", "Engineering", "title"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q in wordSet, not found", w)
		}
	}
}

func TestExtractFromJSON_WordPress(t *testing.T) {
	data := []byte(`{
		"name": "Acme Corp",
		"description": "Enterprise solutions for healthcare",
		"url": "https://acme.example.com",
		"namespaces": ["wp/v2", "custom-api/v1"],
		"authentication": {"cookie": "logged_in"}
	}`)

	wordSet := make(map[string]struct{})
	ExtractFromJSON(data, wordSet)

	expected := []string{"Acme", "Corp", "Enterprise", "healthcare", "authentication", "cookie"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q in wordSet, not found", w)
		}
	}
}

func TestExtractFromJSON_Empty(t *testing.T) {
	wordSet := make(map[string]struct{})
	ExtractFromJSON([]byte(""), wordSet)
	if len(wordSet) != 0 {
		t.Errorf("expected empty wordSet, got %d", len(wordSet))
	}
}

func TestExtractFromJSON_Invalid(t *testing.T) {
	wordSet := make(map[string]struct{})
	ExtractFromJSON([]byte("not json {broken"), wordSet)
	if len(wordSet) != 0 {
		t.Errorf("expected empty wordSet, got %d", len(wordSet))
	}
}

func BenchmarkExtractFromJSON(b *testing.B) {
	data := []byte(`{
		"message": "I just want you to know who I am",
		"author": "sparkle",
		"tags": ["depth", "presence", "intensity"]
	}`)

	for b.Loop() {
		wordSet := make(map[string]struct{})
		ExtractFromJSON(data, wordSet)
	}
}
