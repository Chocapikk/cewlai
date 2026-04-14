package parser

import "testing"

func TestExtractSubtitles_VTT(t *testing.T) {
	vtt := []byte(`WEBVTT

00:00:01.000 --> 00:00:04.000
Welcome to the security presentation

00:00:04.500 --> 00:00:08.000
Today we discuss vulnerability management

00:00:08.500 --> 00:00:12.000
Our team uses <b>Metasploit</b> and <i>Burp Suite</i>
`)

	wordSet := make(map[string]struct{})
	ExtractSubtitles(vtt, wordSet)

	expected := []string{"Welcome", "security", "presentation", "vulnerability", "management", "Metasploit", "Burp", "Suite"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q in wordSet from VTT, not found", w)
		}
	}

	if _, ok := wordSet["00"]; ok {
		t.Error("timestamps should not be in wordSet")
	}
	if _, ok := wordSet["WEBVTT"]; ok {
		t.Error("WEBVTT header should not be in wordSet")
	}
}

func TestExtractSubtitles_SRT(t *testing.T) {
	srt := []byte(`1
00:00:01,000 --> 00:00:04,000
Penetration testing fundamentals

2
00:00:04,500 --> 00:00:08,000
Authentication bypass techniques

3
00:00:08,500 --> 00:00:12,000
Privilege escalation on Linux
`)

	wordSet := make(map[string]struct{})
	ExtractSubtitles(srt, wordSet)

	expected := []string{"Penetration", "testing", "fundamentals", "Authentication", "bypass", "techniques", "Privilege", "escalation", "Linux"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q in wordSet from SRT, not found", w)
		}
	}
}

func TestExtractSubtitles_Empty(t *testing.T) {
	wordSet := make(map[string]struct{})
	ExtractSubtitles([]byte(""), wordSet)
	if len(wordSet) != 0 {
		t.Errorf("expected empty wordSet, got %d", len(wordSet))
	}
}

func BenchmarkExtractSubtitles(b *testing.B) {
	vtt := []byte(`WEBVTT

00:00:01.000 --> 00:00:04.000
People who go all in are rare keep them

00:00:04.500 --> 00:00:08.000
She wants her intensity to be chosen
`)

	for b.Loop() {
		wordSet := make(map[string]struct{})
		ExtractSubtitles(vtt, wordSet)
	}
}
