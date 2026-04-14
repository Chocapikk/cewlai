package crawler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDumpFile(t *testing.T) {
	dir := t.TempDir()

	dumpFile(dir, "docs/readme.txt", []byte("hello"))
	data, err := os.ReadFile(filepath.Join(dir, "docs", "readme.txt"))
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("expected 'hello', got %q", data)
	}
}

func TestDumpFilePathTraversal(t *testing.T) {
	dir := t.TempDir()

	tests := []string{
		"../../etc/passwd",
		"../../../tmp/pwned",
		"foo/../../bar",
		"/etc/shadow",
	}

	for _, path := range tests {
		dumpFile(dir, path, []byte("pwned"))
	}

	// Verify nothing was written outside dir
	entries, _ := filepath.Glob(dir + "/**")
	for _, e := range entries {
		abs, _ := filepath.Abs(e)
		absDir, _ := filepath.Abs(dir)
		if !strings.HasPrefix(abs, absDir+string(filepath.Separator)) {
			t.Errorf("file written outside dump dir: %s", e)
		}
	}

	// Verify /etc/passwd wasn't touched
	if _, err := os.Stat(filepath.Join(dir, "..", "..", "etc", "passwd")); err == nil {
		t.Error("path traversal succeeded: ../../etc/passwd was created")
	}
}

func TestDumpFileSubdirectories(t *testing.T) {
	dir := t.TempDir()

	dumpFile(dir, "a/b/c/deep.txt", []byte("deep"))
	data, err := os.ReadFile(filepath.Join(dir, "a", "b", "c", "deep.txt"))
	if err != nil {
		t.Fatalf("expected deep file: %v", err)
	}
	if string(data) != "deep" {
		t.Errorf("expected 'deep', got %q", data)
	}
}
