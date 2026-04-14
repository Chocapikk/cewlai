package crawler

import "testing"

func TestExtractFromXML_RSS(t *testing.T) {
	rss := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Chocapikk Security Blog</title>
    <description>Vulnerability research and exploit development</description>
    <item>
      <title>CeWL Is Dead</title>
      <description>AI-powered wordlist generator</description>
      <category>Pentest</category>
      <author>Valentin Lobstein</author>
      <pubDate>Mon, 14 Apr 2026 00:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`)

	wordSet := make(map[string]struct{})
	extractFromXML(rss, wordSet)

	expected := []string{"Chocapikk", "Security", "Blog", "Vulnerability", "CeWL", "Dead", "Pentest", "Valentin", "Lobstein"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q in wordSet, not found", w)
		}
	}
}

func TestExtractFromXML_Sitemap(t *testing.T) {
	sitemap := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://chocapikk.com/posts/2026/cewlai/</loc>
    <lastmod>2026-04-14</lastmod>
  </url>
  <url>
    <loc>https://chocapikk.com/about/</loc>
  </url>
</urlset>`)

	wordSet := make(map[string]struct{})
	extractFromXML(sitemap, wordSet)

	if _, ok := wordSet["chocapikk"]; !ok {
		if _, ok2 := wordSet["cewlai"]; !ok2 {
			t.Error("expected URL components in wordSet")
		}
	}
}

func TestExtractFromXML_Atom(t *testing.T) {
	atom := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Security Research</title>
  <entry>
    <title>Exploit Development</title>
    <summary>From zero to exploit developer</summary>
    <author><name>Chocapikk</name></author>
    <category term="offensive"/>
  </entry>
</feed>`)

	wordSet := make(map[string]struct{})
	extractFromXML(atom, wordSet)

	expected := []string{"Security", "Research", "Exploit", "Development", "Chocapikk"}
	for _, w := range expected {
		if _, ok := wordSet[w]; !ok {
			t.Errorf("expected %q in wordSet, not found", w)
		}
	}
}

func TestExtractFromXML_Empty(t *testing.T) {
	wordSet := make(map[string]struct{})
	extractFromXML([]byte(""), wordSet)
	if len(wordSet) != 0 {
		t.Errorf("expected empty wordSet, got %d entries", len(wordSet))
	}
}

func TestExtractFromXML_Invalid(t *testing.T) {
	wordSet := make(map[string]struct{})
	extractFromXML([]byte("not xml at all <broken"), wordSet)
	// Should not panic
}

func BenchmarkExtractFromXML(b *testing.B) {
	rss := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>People who go all in are rare keep them</title>
    <description>May those who want my presence but not my depth never find me</description>
    <item>
      <title>It is hard losing your sparkle for a bit</title>
      <category>resilience</category>
    </item>
  </channel>
</rss>`)

	for b.Loop() {
		wordSet := make(map[string]struct{})
		extractFromXML(rss, wordSet)
	}
}
