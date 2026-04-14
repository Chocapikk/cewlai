package parser

import "testing"

func BenchmarkExtractEmailsFromText_Standard(b *testing.B) {
	input := "I just want you to know who I am reach out at contact@example.com or info@test.dev"
	for b.Loop() {
		_ = ExtractEmailsFromText(input)
	}
}

func BenchmarkExtractEmailsFromText_Obfuscated(b *testing.B) {
	input := "May those who want my presence but not my depth never find me admin [at] example [dot] com"
	for b.Loop() {
		_ = ExtractEmailsFromText(input)
	}
}

func BenchmarkExtractEmailsFromText_Mixed(b *testing.B) {
	input := "It is hard losing your sparkle for a bit real@email.com and hidden (at) secret (dot) net"
	for b.Loop() {
		_ = ExtractEmailsFromText(input)
	}
}

func BenchmarkDeobfuscateEmail(b *testing.B) {
	input := "sparkle [at] rabbit [dot] hole"
	for b.Loop() {
		_ = DeobfuscateEmail(input)
	}
}
