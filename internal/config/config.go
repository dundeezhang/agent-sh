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
	Prompt  PromptConfig  `toml:"prompt"`
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

// PromptConfig controls the shell prompt appearance.
type PromptConfig struct {
	Format      string `toml:"format"`
	RightFormat string `toml:"right_format"`
}

// DefaultPromptFormat is the default prompt format matching the original
// hardcoded prompt: blue "agent-sh", CWD, blue ">".
const DefaultPromptFormat = "\033[1;34magent-sh\033[0m %d\033[1;34m>\033[0m "

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
		Prompt: PromptConfig{
			Format: DefaultPromptFormat,
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

	return cfg, nil
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "agent-sh"), nil
}
