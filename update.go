package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
)

const githubRepo = "Chocapikk/cewlai"

func latestReleaseURL() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
}

func downloadURL(tag, osName, arch string) string {
	ext := ""
	if osName == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/cewlai_%s_%s_%s%s",
		githubRepo, tag, tag, osName, arch, ext,
	)
}

func selfUpdate() {
	log.Println("Checking for updates...")

	latest, err := fetchLatestTag()
	if err != nil {
		log.Fatalf("Failed to check for updates: %v", err)
	}

	current := "v" + version
	if current == latest {
		log.Printf("Already up-to-date (%s)", current)
		return
	}

	log.Printf("New version available: %s (current: %s)", latest, current)

	body, err := downloadRelease(latest)
	if err != nil {
		log.Fatalf("Failed to download update: %v", err)
	}

	if err := replaceBinary(body); err != nil {
		log.Fatalf("Failed to install update: %v", err)
	}

	log.Printf("Updated to %s. Restart cewlai to use the new version.", latest)
	os.Exit(0)
}

func fetchLatestTag() (string, error) {
	resp, err := http.Get(latestReleaseURL())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var result struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.TagName == "" {
		return "", fmt.Errorf("empty tag_name in response")
	}
	return result.TagName, nil
}

func downloadRelease(tag string) ([]byte, error) {
	url := downloadURL(tag, runtime.GOOS, runtime.GOARCH)
	log.Printf("Downloading %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s returned %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func replaceBinary(newBinary []byte) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find executable path: %w", err)
	}

	tmp := exe + ".tmp"
	if err := os.WriteFile(tmp, newBinary, 0o755); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if runtime.GOOS == "windows" {
		old := exe + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			return fmt.Errorf("failed to rename current binary: %w", err)
		}
	} else {
		if err := os.Remove(exe); err != nil {
			return fmt.Errorf("failed to remove current binary: %w", err)
		}
	}

	if err := os.Rename(tmp, exe); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	return nil
}
