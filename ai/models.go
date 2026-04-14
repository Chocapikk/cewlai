package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type modelEntry struct {
	ID string `json:"id"`
}

type modelList struct {
	Data []modelEntry `json:"data"`
}

func ListModels(provider, apiKey, baseURL string) {
	p := strings.ToLower(provider)

	if p == "" {
		fmt.Fprintln(os.Stderr, "Error: -p (provider) is required with --list-models")
		os.Exit(1)
	}

	if preset, ok := providerPresets[p]; ok {
		if baseURL == "" {
			baseURL = preset.baseURL
		}
		if apiKey == "" {
			apiKey = os.Getenv(preset.envKey)
		}
	} else {
		switch p {
		case "anthropic":
			if baseURL == "" {
				baseURL = "https://api.anthropic.com/v1"
			}
			if apiKey == "" {
				apiKey = os.Getenv("ANTHROPIC_API_KEY")
			}
		case "openai":
			if baseURL == "" {
				baseURL = "https://api.openai.com/v1"
			}
			if apiKey == "" {
				apiKey = os.Getenv("OPENAI_API_KEY")
			}
		}
	}

	url := strings.TrimRight(baseURL, "/") + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()

	var models modelList
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Available models for %s:\n\n", provider)
	for _, m := range models.Data {
		fmt.Println(m.ID)
	}
}
