// Package config handles local configuration management.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	mu         sync.RWMutex
	globalCfg  *Config
	configPath string
)

// Config represents the CLI configuration.
type Config struct {
	APIUrl          string            `json:"api_url"`
	Editor          string            `json:"editor,omitempty"`
	RenderFormat    string            `json:"render_format,omitempty"`
	PostVisibility  string            `json:"post_visibility,omitempty"`
	AssetVisibility string            `json:"asset_visibility,omitempty"`
	CustomSettings  map[string]string `json:"custom,omitempty"`
}

// Default returns a config with default values.
func Default() *Config {
	return &Config{
		APIUrl:          "https://api.joinme.sh",
		RenderFormat:    "auto",
		PostVisibility:  "public",
		AssetVisibility: "public",
		CustomSettings:  make(map[string]string),
	}
}

// Load reads the configuration from disk, creating defaults if needed.
func Load() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	if globalCfg != nil {
		return globalCfg, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	mshDir := filepath.Join(homeDir, ".msh")
	if err := os.MkdirAll(mshDir, 0700); err != nil {
		return nil, fmt.Errorf("create .msh directory: %w", err)
	}

	configPath = filepath.Join(mshDir, "config.json")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		globalCfg = Default()
		if err := save(globalCfg); err != nil {
			return nil, fmt.Errorf("save default config: %w", err)
		}
		return globalCfg, nil
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Ensure custom settings map is initialized
	if cfg.CustomSettings == nil {
		cfg.CustomSettings = make(map[string]string)
	}

	globalCfg = &cfg

	// Override from environment
	if apiURL := os.Getenv("MSH_API_URL"); apiURL != "" {
		globalCfg.APIUrl = apiURL
	}

	return globalCfg, nil
}

// save writes the config to disk.
func save(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// Save persists the current config to disk.
func Save() error {
	mu.Lock()
	defer mu.Unlock()

	if globalCfg == nil {
		return fmt.Errorf("no config loaded")
	}

	return save(globalCfg)
}

// Get retrieves a config value by key.
func Get(key string) (string, error) {
	mu.RLock()
	defer mu.RUnlock()

	if globalCfg == nil {
		return "", fmt.Errorf("config not loaded")
	}

	switch key {
	case "api_url":
		return globalCfg.APIUrl, nil
	case "editor":
		return globalCfg.Editor, nil
	case "render.format":
		return globalCfg.RenderFormat, nil
	case "post.visibility":
		return globalCfg.PostVisibility, nil
	case "asset.visibility":
		return globalCfg.AssetVisibility, nil
	default:
		// Check custom settings
		if val, ok := globalCfg.CustomSettings[key]; ok {
			return val, nil
		}
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Set updates a config value by key.
func Set(key, value string) error {
	mu.Lock()
	defer mu.Unlock()

	if globalCfg == nil {
		return fmt.Errorf("config not loaded")
	}

	switch key {
	case "api_url":
		globalCfg.APIUrl = value
	case "editor":
		globalCfg.Editor = value
	case "render.format":
		globalCfg.RenderFormat = value
	case "post.visibility":
		globalCfg.PostVisibility = value
	case "asset.visibility":
		globalCfg.AssetVisibility = value
	default:
		// Store in custom settings
		globalCfg.CustomSettings[key] = value
	}

	return save(globalCfg)
}

// List returns all config key-value pairs.
func List() (map[string]string, error) {
	mu.RLock()
	defer mu.RUnlock()

	if globalCfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	result := make(map[string]string)
	result["api_url"] = globalCfg.APIUrl
	result["editor"] = globalCfg.Editor
	result["render.format"] = globalCfg.RenderFormat
	result["post.visibility"] = globalCfg.PostVisibility
	result["asset.visibility"] = globalCfg.AssetVisibility

	// Add custom settings
	for k, v := range globalCfg.CustomSettings {
		result[k] = v
	}

	return result, nil
}

// GetAPIUrl returns the configured API URL.
func GetAPIUrl() string {
	mu.RLock()
	defer mu.RUnlock()

	if globalCfg == nil {
		return "https://api.joinme.sh"
	}

	return globalCfg.APIUrl
}
