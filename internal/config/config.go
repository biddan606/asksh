package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Backend BackendConfig `toml:"backend"`
	Ollama  OllamaConfig  `toml:"ollama"`
	OpenAI  OpenAIConfig  `toml:"openai"`
	Safety  SafetyConfig  `toml:"safety"`
	History HistoryConfig `toml:"history"`
}

type BackendConfig struct {
	Default string `toml:"default"`
}

type OllamaConfig struct {
	Host  string `toml:"host"`
	Model string `toml:"model"`
}

type OpenAIConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

type SafetyConfig struct {
	RequireConfirmation bool `toml:"require_confirmation"`
	ExtraLLMCheck       bool `toml:"extra_llm_check"`
}

type HistoryConfig struct {
	Enable bool `toml:"enable"`
}

func DefaultConfig() Config {
	return Config{
		Backend: BackendConfig{Default: "ollama"},
		Ollama:  OllamaConfig{Host: "http://localhost:11434", Model: "qwen2.5-coder:7b"},
		OpenAI:  OpenAIConfig{Model: "gpt-4o-mini"},
		Safety:  SafetyConfig{RequireConfirmation: true, ExtraLLMCheck: true},
		History: HistoryConfig{Enable: false},
	}
}

func ConfigPath() string {
	if dir, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok && dir != "" {
		return filepath.Join(dir, "asksh", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "asksh", "config.toml")
}

func Load(path string) (Config, error) {
	cfg := DefaultConfig()
	_, err := toml.DecodeFile(path, &cfg)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err != nil {
		return cfg, fmt.Errorf("decode %s: %w", path, err)
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	encErr := toml.NewEncoder(f).Encode(cfg)
	if cerr := f.Close(); encErr == nil {
		return cerr
	}
	return encErr
}
