package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	API     APIConfig     `toml:"api"`
	Context ContextConfig `toml:"context"`
	Shell   ShellConfig   `toml:"shell"`
}

type ShellConfig struct {
	EditMode string `toml:"edit_mode"`
}

type APIConfig struct {
	Provider string `toml:"provider"`
	Key      string `toml:"key"`
	Model    string `toml:"model"`
	BaseURL  string `toml:"base_url"`
}

type ContextConfig struct {
	HistorySize int  `toml:"history_size"`
	IncludeGit  bool `toml:"include_git"`
}

func DefaultConfig() *Config {
	return &Config{
		API: APIConfig{
			Provider: "anthropic",
			Model:    "sonnet",
		},
		Context: ContextConfig{
			HistorySize: 20,
			IncludeGit:  true,
		},
		Shell: ShellConfig{
			EditMode: "emacs",
		},
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	configDir, err := configDir()
	if err != nil {
		return cfg, nil
	}

	configPath := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		if _, err := toml.DecodeFile(configPath, cfg); err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}
	}

	// Env vars override config file
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" && cfg.API.Key == "" {
		cfg.API.Key = key
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" && cfg.API.Key == "" && cfg.API.Provider == "openai" {
		cfg.API.Key = key
	}
	if provider := os.Getenv("AGENT_SH_PROVIDER"); provider != "" {
		cfg.API.Provider = provider
	}
	if model := os.Getenv("AGENT_SH_MODEL"); model != "" {
		cfg.API.Model = model
	}
	if editMode := os.Getenv("AGENT_SH_EDIT_MODE"); editMode != "" {
		cfg.Shell.EditMode = editMode
	}

	return cfg, nil
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "agent-sh"), nil
}
