package words

import (
	"encoding/json"
	"os"
	"strings"
	"unicode"
)

type MutateConfig struct {
	Leet        map[string]string `json:"leet"`
	Suffixes    []string          `json:"suffixes"`
	Prefixes    []string          `json:"prefixes"`
	Capitalize  bool              `json:"capitalize"`
	Uppercase   bool              `json:"uppercase"`
	Lowercase   bool              `json:"lowercase"`
	Reverse     bool              `json:"reverse"`
	LeetEnabled bool              `json:"leet_enabled"`
	MinLength   int               `json:"min_length"`
	MaxLength   int               `json:"max_length"`
}

func DefaultMutateConfig() MutateConfig {
	return MutateConfig{
		Leet: map[string]string{
			"a": "4", "e": "3", "i": "1", "o": "0",
			"s": "5", "t": "7", "g": "9", "z": "2",
		},
		Suffixes: []string{
			"1", "12", "123", "1234",
			"!", "!!", "@", "#", "$",
			"01", "69", "99",
			"2024", "2025", "2026",
		},
		Prefixes:    []string{},
		Capitalize:  true,
		Uppercase:   true,
		Lowercase:   true,
		Reverse:     true,
		LeetEnabled: true,
		MinLength:   0,
		MaxLength:   0,
	}
}

func LoadMutateConfig(path string) (MutateConfig, error) {
	cfg := DefaultMutateConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func MutateWords(input []string, cfg MutateConfig) []string {
	seen := make(map[string]struct{})
	var out []string

	add := func(w string) {
		if w == "" {
			return
		}
		if cfg.MinLength > 0 && len(w) < cfg.MinLength {
			return
		}
		if cfg.MaxLength > 0 && len(w) > cfg.MaxLength {
			return
		}
		if _, exists := seen[w]; exists {
			return
		}
		seen[w] = struct{}{}
		out = append(out, w)
	}

	for _, w := range input {
		add(w)
		applyBaseMutations(w, cfg, add)
		applySuffixes(w, cfg, add)
		applyPrefixes(w, cfg, add)
	}

	return out
}

func applyBaseMutations(w string, cfg MutateConfig, add func(string)) {
	if cfg.Lowercase {
		add(strings.ToLower(w))
	}
	if cfg.Uppercase {
		add(strings.ToUpper(w))
	}
	if cfg.Capitalize {
		add(capitalize(w))
	}
	if cfg.Reverse {
		add(reverseStr(w))
	}
	if cfg.LeetEnabled {
		add(leet(w, cfg.Leet))
		add(leet(strings.ToLower(w), cfg.Leet))
	}
}

func applySuffixes(w string, cfg MutateConfig, add func(string)) {
	variants := []string{w}
	if cfg.Lowercase {
		variants = append(variants, strings.ToLower(w))
	}
	if cfg.Capitalize {
		variants = append(variants, capitalize(w))
	}

	for _, v := range variants {
		for _, suffix := range cfg.Suffixes {
			add(v + suffix)
		}
	}
}

func applyPrefixes(w string, cfg MutateConfig, add func(string)) {
	for _, prefix := range cfg.Prefixes {
		add(prefix + w)
		if cfg.Capitalize {
			add(prefix + capitalize(w))
		}
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}

func reverseStr(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func leet(s string, leetMap map[string]string) string {
	changed := false
	var b strings.Builder
	for _, r := range s {
		if rep, ok := leetMap[strings.ToLower(string(r))]; ok {
			b.WriteString(rep)
			changed = true
		} else {
			b.WriteRune(r)
		}
	}
	if !changed {
		return ""
	}
	return b.String()
}
