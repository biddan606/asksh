package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/biddan606/asksh/internal/config"
)

func TestWizardAcceptsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Simulate user pressing Enter for every prompt (accept defaults).
	input := strings.Repeat("\n", 20)
	var out strings.Builder

	if err := config.RunWizard(strings.NewReader(input), &out, path); err != nil {
		t.Fatalf("RunWizard error: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	def := config.DefaultConfig()

	if cfg.Backend.Default != def.Backend.Default {
		t.Errorf("backend.default: got %q, want %q", cfg.Backend.Default, def.Backend.Default)
	}
	if cfg.Ollama.Host != def.Ollama.Host {
		t.Errorf("ollama.host: got %q, want %q", cfg.Ollama.Host, def.Ollama.Host)
	}
	if cfg.Ollama.Model != def.Ollama.Model {
		t.Errorf("ollama.model: got %q, want %q", cfg.Ollama.Model, def.Ollama.Model)
	}
	if cfg.OpenAI.Model != def.OpenAI.Model {
		t.Errorf("openai.model: got %q, want %q", cfg.OpenAI.Model, def.OpenAI.Model)
	}
	if cfg.Safety.RequireConfirmation != def.Safety.RequireConfirmation {
		t.Errorf("safety.require_confirmation: got %v, want %v", cfg.Safety.RequireConfirmation, def.Safety.RequireConfirmation)
	}
	if cfg.Safety.ExtraLLMCheck != def.Safety.ExtraLLMCheck {
		t.Errorf("safety.extra_llm_check: got %v, want %v", cfg.Safety.ExtraLLMCheck, def.Safety.ExtraLLMCheck)
	}
	if cfg.History.Enable != def.History.Enable {
		t.Errorf("history.enable: got %v, want %v", cfg.History.Enable, def.History.Enable)
	}
}

func TestWizardCustomValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Provide custom answers for each prompt in order:
	// backend.default, ollama.host, ollama.model, openai.api_key, openai.model,
	// safety.require_confirmation, safety.extra_llm_check, history.enable
	input := strings.Join([]string{
		"openai",
		"http://custom:11434",
		"llama3",
		"sk-mykey",
		"gpt-4",
		"n",
		"n",
		"y",
		"",
	}, "\n")

	if err := config.RunWizard(strings.NewReader(input), &strings.Builder{}, path); err != nil {
		t.Fatalf("RunWizard error: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Backend.Default != "openai" {
		t.Errorf("backend.default: got %q, want openai", cfg.Backend.Default)
	}
	if cfg.Ollama.Host != "http://custom:11434" {
		t.Errorf("ollama.host: got %q, want http://custom:11434", cfg.Ollama.Host)
	}
	if cfg.Ollama.Model != "llama3" {
		t.Errorf("ollama.model: got %q, want llama3", cfg.Ollama.Model)
	}
	if cfg.OpenAI.APIKey != "sk-mykey" {
		t.Errorf("openai.api_key: got %q, want sk-mykey", cfg.OpenAI.APIKey)
	}
	if cfg.OpenAI.Model != "gpt-4" {
		t.Errorf("openai.model: got %q, want gpt-4", cfg.OpenAI.Model)
	}
	if cfg.Safety.RequireConfirmation {
		t.Error("safety.require_confirmation: got true, want false")
	}
	if cfg.Safety.ExtraLLMCheck {
		t.Error("safety.extra_llm_check: got true, want false")
	}
	if !cfg.History.Enable {
		t.Error("history.enable: got false, want true")
	}
}

func TestWizardPromptsShowDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	input := strings.Repeat("\n", 20)
	var out strings.Builder

	if err := config.RunWizard(strings.NewReader(input), &out, path); err != nil {
		t.Fatalf("RunWizard error: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"ollama",
		"http://localhost:11434",
		"qwen2.5-coder:7b",
		"gpt-4o-mini",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q; output: %s", want, output)
		}
	}
}
