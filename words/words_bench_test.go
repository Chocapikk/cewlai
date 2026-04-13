package words

import "testing"

var benchText = "People who go all in are rare keep them she wants her intensity to be chosen not her silence"

func BenchmarkNormalizeAndSplit(b *testing.B) {
	for b.Loop() {
		_ = NormalizeAndSplit(benchText)
	}
}

func BenchmarkFilterWords(b *testing.B) {
	input := NormalizeAndSplit(benchText)
	b.ResetTimer()
	for b.Loop() {
		_ = FilterWords(input, 3, 10, true)
	}
}

func BenchmarkFilterWords_NoNumbers(b *testing.B) {
	input := []string{"hello", "world", "test123", "foo42", "bar", "abc", "x1y"}
	b.ResetTimer()
	for b.Loop() {
		_ = FilterWords(input, 3, 0, false)
	}
}

func BenchmarkLowercaseWords(b *testing.B) {
	input := []string{"Hello", "WORLD", "FoO", "BaR", "Test", "CASE", "MiXeD"}
	b.ResetTimer()
	for b.Loop() {
		_ = LowercaseWords(input)
	}
}

func BenchmarkDeduplicateWords(b *testing.B) {
	set1 := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	set2 := []string{"charlie", "delta", "foxtrot", "golf", "hotel"}
	b.ResetTimer()
	for b.Loop() {
		_ = DeduplicateWords(set1, set2)
	}
}

func BenchmarkDeduplicateWords_Large(b *testing.B) {
	input := make([]string, 1000)
	for i := range input {
		input[i] = benchText
	}
	words := NormalizeAndSplit(benchText)
	b.ResetTimer()
	for b.Loop() {
		_ = DeduplicateWords(words, words, words)
	}
}

func BenchmarkCountWords(b *testing.B) {
	input := NormalizeAndSplit(benchText)
	b.ResetTimer()
	for b.Loop() {
		_ = CountWords(input)
	}
}

func BenchmarkFormatWithCounts(b *testing.B) {
	input := NormalizeAndSplit(benchText)
	b.ResetTimer()
	for b.Loop() {
		_ = FormatWithCounts(input)
	}
}

func BenchmarkGenerateGroups(b *testing.B) {
	input := NormalizeAndSplit(benchText)
	b.ResetTimer()
	for b.Loop() {
		_ = GenerateGroups(input, 3)
	}
}
