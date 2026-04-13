package words

import (
	"testing"
)

func TestNormalizeAndSplit(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"hello  world", []string{"hello", "world"}},
		{"hello-world", []string{"hello-world"}},
		{"it's a test", []string{"it's", "a", "test"}},
		{"foo@bar.com", []string{"foo", "bar", "com"}},
		{"", nil},
		{"   ", nil},
		{"one", []string{"one"}},
	}
	for _, tt := range tests {
		got := NormalizeAndSplit(tt.input)
		if !sliceEqual(got, tt.want) {
			t.Errorf("NormalizeAndSplit(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeAndSplit_NoAlloc(t *testing.T) {
	input := "hello world foo bar baz"
	avg := testing.AllocsPerRun(100, func() {
		_ = NormalizeAndSplit(input)
	})
	if avg > 10 {
		t.Errorf("NormalizeAndSplit: %.1f allocs, want <= 10", avg)
	}
}

func TestFilterWords(t *testing.T) {
	words := []string{"ab", "abc", "abcd", "abcde", "123", "ab1c"}

	got := FilterWords(words, 3, 4, true)
	want := []string{"abc", "abcd", "123", "ab1c"}
	if !sliceEqual(got, want) {
		t.Errorf("FilterWords(min=3,max=4,nums=true) = %v, want %v", got, want)
	}

	got = FilterWords(words, 3, 4, false)
	want = []string{"abc", "abcd"}
	if !sliceEqual(got, want) {
		t.Errorf("FilterWords(min=3,max=4,nums=false) = %v, want %v", got, want)
	}

	got = FilterWords(words, 3, 0, true)
	want = []string{"abc", "abcd", "abcde", "123", "ab1c"}
	if !sliceEqual(got, want) {
		t.Errorf("FilterWords(min=3,max=0,nums=true) = %v, want %v", got, want)
	}
}

func TestFilterWords_NoAlloc(t *testing.T) {
	input := []string{"abc", "defgh", "ij", "klmno", "pq", "rstuv"}
	avg := testing.AllocsPerRun(100, func() {
		_ = FilterWords(input, 3, 5, true)
	})
	if avg > 2 {
		t.Errorf("FilterWords: %.1f allocs, want <= 2", avg)
	}
}

func TestLowercaseWords(t *testing.T) {
	got := LowercaseWords([]string{"Hello", "WORLD", "fOo"})
	want := []string{"hello", "world", "foo"}
	if !sliceEqual(got, want) {
		t.Errorf("LowercaseWords = %v, want %v", got, want)
	}
}

func TestDeduplicateWords(t *testing.T) {
	got := DeduplicateWords([]string{"b", "a", "c"}, []string{"a", "d", "b"})
	want := []string{"a", "b", "c", "d"}
	if !sliceEqual(got, want) {
		t.Errorf("DeduplicateWords = %v, want %v", got, want)
	}
}

func TestDeduplicateWords_CaseSensitive(t *testing.T) {
	got := DeduplicateWords([]string{"Admin", "admin", "ADMIN"})
	want := []string{"ADMIN", "Admin", "admin"}
	if !sliceEqual(got, want) {
		t.Errorf("DeduplicateWords case sensitive = %v, want %v", got, want)
	}
}

func TestCountWords(t *testing.T) {
	counts := CountWords([]string{"a", "b", "a", "c", "a", "b"})
	if counts["a"] != 3 || counts["b"] != 2 || counts["c"] != 1 {
		t.Errorf("CountWords = %v", counts)
	}
}

func TestGenerateGroups(t *testing.T) {
	got := GenerateGroups([]string{"a", "b", "c", "d"}, 2)
	want := []string{"a b", "b c", "c d"}
	if !sliceEqual(got, want) {
		t.Errorf("GenerateGroups(2) = %v, want %v", got, want)
	}

	got = GenerateGroups([]string{"a", "b"}, 3)
	if got != nil {
		t.Errorf("GenerateGroups(3) with 2 words = %v, want nil", got)
	}

	got = GenerateGroups([]string{"a", "b"}, 1)
	if got != nil {
		t.Errorf("GenerateGroups(1) = %v, want nil", got)
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
