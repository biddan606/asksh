package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/biddan606/asksh/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.Backend.Default != "ollama" {
		t.Errorf("default backend = %q, want ollama", cfg.Backend.Default)
	}
	if cfg.Ollama.Host != "http://localhost:11434" {
		t.Errorf("ollama host = %q, want http://localhost:11434", cfg.Ollama.Host)
	}
	if cfg.Ollama.Model != "qwen2.5-coder:7b" {
		t.Errorf("ollama model = %q, want qwen2.5-coder:7b", cfg.Ollama.Model)
	}
	if cfg.OpenAI.Model != "gpt-4o-mini" {
		t.Errorf("openai model = %q, want gpt-4o-mini", cfg.OpenAI.Model)
	}
	if !cfg.Safety.RequireConfirmation {
		t.Error("require_confirmation should default to true")
	}
	if !cfg.Safety.ExtraLLMCheck {
		t.Error("extra_llm_check should default to true")
	}
	if cfg.History.Enable {
		t.Error("history.enable should default to false")
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	original := config.DefaultConfig()
	original.Backend.Default = "openai"
	original.OpenAI.APIKey = "sk-test"
	original.OpenAI.Model = "gpt-4"

	if err := config.Save(path, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Backend.Default != original.Backend.Default {
		t.Errorf("backend.default: got %q, want %q", loaded.Backend.Default, original.Backend.Default)
	}
	if loaded.OpenAI.APIKey != original.OpenAI.APIKey {
		t.Errorf("openai.api_key: got %q, want %q", loaded.OpenAI.APIKey, original.OpenAI.APIKey)
	}
	if loaded.OpenAI.Model != original.OpenAI.Model {
		t.Errorf("openai.model: got %q, want %q", loaded.OpenAI.Model, original.OpenAI.Model)
	}
	if loaded.Ollama.Host != original.Ollama.Host {
		t.Errorf("ollama.host: got %q, want %q", loaded.Ollama.Host, original.Ollama.Host)
	}
}

func TestSaveFileMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := config.Save(path, config.DefaultConfig()); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.toml")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load of missing file failed: %v", err)
	}
	def := config.DefaultConfig()
	if cfg.Backend.Default != def.Backend.Default {
		t.Errorf("expected default config for missing file, got backend=%q", cfg.Backend.Default)
	}
}

func TestConfigPathIsUnderDotConfig(t *testing.T) {
	path := config.ConfigPath()
	if filepath.Base(filepath.Dir(path)) != "asksh" {
		t.Errorf("config path %q should be under .../asksh/", path)
	}
	if filepath.Base(path) != "config.toml" {
		t.Errorf("config filename should be config.toml, got %q", filepath.Base(path))
	}
}
