package crawler

import (
	"testing"
)

func TestExtractEmailsFromText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"standard", "contact us at admin@example.com for info", []string{"admin@example.com"}},
		{"none", "no emails here", nil},
		{"uppercase", "USER@DOMAIN.COM is uppercase", []string{"user@domain.com"}},
		{"multiple", "a@b.com and c@d.org here", []string{"a@b.com", "c@d.org"}},
		{"plus tag", "edge.case+tag@sub.domain.co works", []string{"edge.case+tag@sub.domain.co"}},
		{"at obfuscation", "contact user [at] example [dot] com", []string{"user@example.com"}},
		{"at parens", "email: admin (at) test (dot) org", []string{"admin@test.org"}},
		{"at angle", "info <at> company <dot> net", []string{"info@company.net"}},
		{"at curly", "hello {at} world {dot} com", []string{"hello@world.com"}},
		{"AT uppercase", "user AT domain DOT com", []string{"user@domain.com"}},
		{"mixed", "real@email.com and fake [at] obfuscated [dot] org", []string{"real@email.com", "fake@obfuscated.org"}},
		{"no dupe", "same@email.com and same@email.com", []string{"same@email.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEmailsFromText(tt.input)
			if !sliceEqual(got, tt.want) {
				t.Errorf("extractEmailsFromText(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractEmailsFromText_NoAlloc(t *testing.T) {
	input := "reach us at info@company.com or support@company.com"
	avg := testing.AllocsPerRun(100, func() {
		_ = extractEmailsFromText(input)
	})
	if avg > 25 {
		t.Errorf("extractEmailsFromText: %.1f allocs, want <= 25", avg)
	}
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
