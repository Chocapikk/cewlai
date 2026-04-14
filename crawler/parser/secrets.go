package parser

import (
	"context"
	"strings"
	"unsafe"

	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/engine/defaults"
)

// unsafeString creates a string from a byte slice without copying.
// The caller must ensure the byte slice is not modified while the string is in use.
func unsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

type SecretFinding struct {
	DetectorName string `json:"detector"`
	Raw          string `json:"raw"`
	Redacted     string `json:"redacted,omitempty"`
	Source       string `json:"source,omitempty"`
}

type SecretScanner struct {
	detectors []detectors.Detector
}

func NewSecretScanner() *SecretScanner {
	return &SecretScanner{
		detectors: defaults.DefaultDetectors(),
	}
}

// Scan checks data for secrets using trufflehog detectors.
// Accepts []byte directly to avoid unnecessary string/byte conversions.
func (s *SecretScanner) Scan(data []byte, source string) []SecretFinding {
	if len(data) < 8 {
		return nil
	}

	ctx := context.Background()
	dataStr := unsafeString(data)
	var findings []SecretFinding

	for _, detector := range s.detectors {
		if !hasKeyword(dataStr, detector.Keywords()) {
			continue
		}

		results, err := detector.FromData(ctx, false, data)
		if err != nil {
			continue
		}

		for _, res := range results {
			findings = append(findings, SecretFinding{
				DetectorName: res.DetectorType.String(),
				Raw:          string(res.Raw),
				Redacted:     res.Redacted,
				Source:       source,
			})
		}
	}

	return findings
}

func hasKeyword(data string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	lower := strings.ToLower(data)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}
