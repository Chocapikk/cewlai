package parser

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"regexp"
	"strings"
	"sync"
)

var (
	DocumentExts = regexp.MustCompile(`(?i)\.(docx|xlsx|pptx|dotx|potx|ppsx)$`)
	PdfExt       = regexp.MustCompile(`(?i)\.pdf$`)

	pdfPatterns = []*regexp.Regexp{
		regexp.MustCompile(`/Author\s*\(([^)]+)\)`),
		regexp.MustCompile(`/Creator\s*\(([^)]+)\)`),
		regexp.MustCompile(`<dc:creator>([^<]+)</dc:creator>`),
		regexp.MustCompile(`<pdf:Author>([^<]+)</pdf:Author>`),
	}

	officePatterns = []*regexp.Regexp{
		regexp.MustCompile(`<dc:creator>([^<]+)</dc:creator>`),
		regexp.MustCompile(`<cp:lastModifiedBy>([^<]+)</cp:lastModifiedBy>`),
	}
)

func ExtractPDFMetadata(body []byte, mu *sync.Mutex, metaSet map[string]struct{}, verbose bool, reqURL string) {
	names := matchAll(string(body), pdfPatterns)

	mu.Lock()
	defer mu.Unlock()
	addMetadata(metaSet, names, verbose, "PDF", reqURL)
}

func ExtractOfficeMetadata(body []byte, mu *sync.Mutex, metaSet map[string]struct{}, verbose bool, reqURL string) {
	content, err := readZipEntry(body, "docProps/core.xml")
	if err != nil {
		if verbose {
			log.Printf("Failed to read Office metadata from %s: %v", reqURL, err)
		}
		return
	}

	names := matchAll(content, officePatterns)

	mu.Lock()
	defer mu.Unlock()
	addMetadata(metaSet, names, verbose, "Office", reqURL)
}

func matchAll(content string, patterns []*regexp.Regexp) []string {
	var results []string
	for _, re := range patterns {
		for _, match := range re.FindAllStringSubmatch(content, -1) {
			if name := strings.TrimSpace(match[1]); name != "" {
				results = append(results, name)
			}
		}
	}
	return results
}

func addMetadata(metaSet map[string]struct{}, names []string, verbose bool, source, reqURL string) {
	for _, name := range names {
		if verbose {
			log.Printf("%s metadata from %s: %s", source, reqURL, name)
		}
		metaSet[name] = struct{}{}
	}
}

func readZipEntry(body []byte, entryName string) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", err
	}

	for _, f := range reader.File {
		if f.Name != entryName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer func() { _ = rc.Close() }()
		data, err := io.ReadAll(rc)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return "", nil
}
