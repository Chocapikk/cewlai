package crawler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const cacheDir = ".cewlai-cache"

type cachedResult struct {
	Result    CrawlResult `json:"result"`
	Timestamp time.Time   `json:"timestamp"`
}

func cacheKey(url string, depth int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", url, depth)))
	return fmt.Sprintf("%x", h[:16])
}

func loadCache(url string, depth int, ttl time.Duration) (*CrawlResult, bool) {
	path := filepath.Join(cacheDir, cacheKey(url, depth)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var cached cachedResult
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}

	if time.Since(cached.Timestamp) > ttl {
		return nil, false
	}

	return &cached.Result, true
}

func saveCache(url string, depth int, result *CrawlResult) {
	_ = os.MkdirAll(cacheDir, 0755)
	path := filepath.Join(cacheDir, cacheKey(url, depth)+".json")

	cached := cachedResult{
		Result:    *result,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}
