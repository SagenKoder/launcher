package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config captures launcher configuration from config.yaml.
type Config struct {
	Chat  ChatConfig   `yaml:"chat"`
	Links []LinkConfig `yaml:"links"`
}

// ChatConfig contains AI chat plugin configuration.
type ChatConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

// LinkConfig contains a configured link plugin.
type LinkConfig struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	Icon string `yaml:"icon"`
	// Replacement defines a substring in the URL that will be replaced with
	// user-provided, URL-encoded input before launching the browser. When empty,
	// the URL is opened immediately without prompting for input.
	Replacement string `yaml:"replacement"`
}

var (
	loadOnce sync.Once
	loaded   Config
	loadErr  error
	pathUsed string
)

// Load reads configuration from the first existing config file among a set of
// candidate paths. The result is cached for subsequent callers.
func Load() (Config, error) {
	loadOnce.Do(func() {
		path, err := findConfigPath()
		if err != nil {
			loadErr = err
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			loadErr = fmt.Errorf("read config %q: %w", path, err)
			return
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			loadErr = fmt.Errorf("parse config %q: %w", path, err)
			return
		}
		loaded = cfg
		pathUsed = path
	})
	return loaded, loadErr
}

// Path returns the path of the configuration file that was loaded. It returns
// an empty string if Load hasn't succeeded yet.
func Path() string {
	return pathUsed
}

func findConfigPath() (string, error) {
	paths := candidatePaths()
	for _, candidate := range paths {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("config file not found in any of: %s", strings.Join(paths, ", "))
}

func candidatePaths() []string {
	var paths []string
	if explicit := os.Getenv("LAUNCHER_CONFIG"); explicit != "" {
		paths = append(paths, explicit)
	}
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, "config.yaml"))
	}
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		paths = append(paths, filepath.Join(configHome, "launcher", "config.yaml"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "launcher", "config.yaml"))
	}
	paths = append(paths, "/etc/launcher/config.yaml")
	return paths
}
