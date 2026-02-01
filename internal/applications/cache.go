package applications

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type AppCache struct {
	Apps      []Application `json:"apps"`
	Timestamp time.Time     `json:"timestamp"`
}

func CachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Caches", "Launcher", "apps.json")
}

func LoadCached() ([]Application, bool) {
	path := CachePath()
	if path == "" {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var cache AppCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}

	return cache.Apps, true
}

func SaveCache(apps []Application) error {
	path := CachePath()
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	cache := AppCache{
		Apps:      apps,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
