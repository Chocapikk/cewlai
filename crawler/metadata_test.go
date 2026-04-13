package crawler

import (
	"archive/zip"
	"bytes"
	"sync"
	"testing"
)

func TestExtractPDFMetadata(t *testing.T) {
	pdf := []byte(`%PDF-1.4
/Author (John Doe)
/Creator (Microsoft Word)
<dc:creator>Jane Smith</dc:creator>
<pdf:Author>Bob Wilson</pdf:Author>
`)
	var mu sync.Mutex
	metaSet := make(map[string]struct{})

	extractPDFMetadata(pdf, &mu, metaSet, false, "test.pdf")

	expected := []string{"John Doe", "Microsoft Word", "Jane Smith", "Bob Wilson"}
	for _, name := range expected {
		if _, ok := metaSet[name]; !ok {
			t.Errorf("expected %q in metadata, got %v", name, metaSet)
		}
	}
}

func TestExtractOfficeMetadata(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	f, _ := w.Create("docProps/core.xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<cp:coreProperties>
  <dc:creator>Alice Author</dc:creator>
  <cp:lastModifiedBy>Bob Editor</cp:lastModifiedBy>
</cp:coreProperties>`))

	w.Close()

	var mu sync.Mutex
	metaSet := make(map[string]struct{})

	extractOfficeMetadata(buf.Bytes(), &mu, metaSet, false, "test.docx")

	if _, ok := metaSet["Alice Author"]; !ok {
		t.Errorf("expected 'Alice Author' in metadata, got %v", metaSet)
	}
	if _, ok := metaSet["Bob Editor"]; !ok {
		t.Errorf("expected 'Bob Editor' in metadata, got %v", metaSet)
	}
}

func TestExtractPDFMetadata_NoAlloc(t *testing.T) {
	pdf := []byte(`%PDF-1.4 /Author (Test User) /Creator (TestApp)`)
	var mu sync.Mutex
	metaSet := make(map[string]struct{})

	avg := testing.AllocsPerRun(100, func() {
		extractPDFMetadata(pdf, &mu, metaSet, false, "test.pdf")
	})
	if avg > 10 {
		t.Errorf("extractPDFMetadata: %.1f allocs, want <= 10", avg)
	}
}
