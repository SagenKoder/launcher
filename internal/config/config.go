package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	configMu sync.RWMutex
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

// DefaultPath returns the default configuration file path for saving.
// On macOS this is ~/Library/Application Support/Launcher/config.yaml.
func DefaultPath() string {
	if pathUsed != "" {
		return pathUsed
	}
	if explicit := os.Getenv("LAUNCHER_CONFIG"); explicit != "" {
		return explicit
	}
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Application Support", "Launcher", "config.yaml")
		}
	}
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "launcher", "config.yaml")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "launcher", "config.yaml")
	}
	return "config.yaml"
}

// Save writes the configuration to disk. If a config was previously loaded,
// it saves to that path; otherwise it uses DefaultPath().
func Save(cfg Config) error {
	path := DefaultPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	// Update the cached config
	configMu.Lock()
	loaded = cfg
	if pathUsed == "" {
		pathUsed = path
	}
	configMu.Unlock()
	return nil
}

// Update replaces the cached config in memory without writing to disk.
func Update(cfg Config) {
	configMu.Lock()
	loaded = cfg
	configMu.Unlock()
}

// Get returns the currently loaded config. It must be called after Load().
func Get() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return loaded
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
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			paths = append(paths, filepath.Join(home, "Library", "Application Support", "Launcher", "config.yaml"))
		}
		paths = append(paths, filepath.Join("/Library", "Application Support", "Launcher", "config.yaml"))
	}
	paths = append(paths, "/etc/launcher/config.yaml")
	return paths
}
