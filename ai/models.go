package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
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
		case "opencode":
			// opencode lists its own providers/models over /config/providers.
			listOpenCodeModels(apiKey, baseURL)
			return
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

// openCodeProvidersResp is the shaped subset of opencode's
// GET /config/providers response we care about.
type openCodeProvidersResp struct {
	Providers []struct {
		ID     string                     `json:"id"`
		Models map[string]openCodeModelID `json:"models"`
	} `json:"providers"`
	// default maps providerID -> modelID.
	Default map[string]string `json:"default"`
}

type openCodeModelID struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// openCodeListTimeout bounds the model listing call. Unlike a message, this is
// a plain lookup, so it should never take long.
const openCodeListTimeout = 30 * time.Second

// listOpenCodeModels prints the providers/models exposed by a local opencode
// server via GET /config/providers.
func listOpenCodeModels(apiKey, baseURL string) {
	if baseURL == "" {
		baseURL = DefaultOpenCodeBaseURL
	}
	url := strings.TrimRight(baseURL, "/") + "/config/providers"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	setOpenCodeAuth(req, opencodeServerPassword(apiKey))

	client := &http.Client{Timeout: openCodeListTimeout}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to opencode at %s: %v\n", baseURL, err)
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()

	// Without this the 401 from a password-protected server surfaces as a
	// confusing JSON parse error instead of an auth problem.
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Fprintf(os.Stderr, "Error: opencode at %s requires a password. Pass it with --api-key or set %s.\n",
				baseURL, OpenCodeServerPasswordEnv)
		} else {
			fmt.Fprintf(os.Stderr, "Error: opencode at %s returned status %d\n", baseURL, resp.StatusCode)
		}
		os.Exit(1)
	}

	var data openCodeProvidersResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&data); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing opencode response: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Available models for opencode (%s):\n\n", baseURL)
	shown := 0
	for _, pr := range data.Providers {
		if len(pr.Models) == 0 {
			continue
		}
		fmt.Printf("%s:\n", pr.ID)
		ids := make([]string, 0, len(pr.Models))
		for id := range pr.Models {
			ids = append(ids, id)
		}
		// Map iteration order is random, sort so successive runs match.
		sort.Strings(ids)
		for _, id := range ids {
			fmt.Printf("  %s/%s\n", pr.ID, id)
			shown++
		}
	}
	if shown == 0 {
		fmt.Fprintln(os.Stderr, "(no provider model list returned; make sure opencode serve is running and providers are configured)")
	}
	if def, ok := data.Default["opencode"]; ok {
		fmt.Fprintf(os.Stderr, "\nDefault (opencode) model: %s\n", def)
	}
}
