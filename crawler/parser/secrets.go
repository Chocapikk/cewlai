package parser

import (
	"context"
	"strings"

	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/engine/defaults"
)

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

func (s *SecretScanner) Scan(data, source string) []SecretFinding {
	if len(data) < 8 {
		return nil
	}

	ctx := context.Background()
	dataBytes := []byte(data)
	var findings []SecretFinding

	for _, detector := range s.detectors {
		if !hasKeyword(data, detector.Keywords()) {
			continue
		}

		results, err := detector.FromData(ctx, false, dataBytes)
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
