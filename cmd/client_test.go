package cmd

import (
	"testing"

	"github.com/biddan606/asksh/internal/config"
)

func TestNewClientForBackendOllama(t *testing.T) {
	cfg := config.DefaultConfig()
	client, err := newClientForBackend(cfg, "ollama")
	if err != nil {
		t.Fatalf("expected no error for ollama backend, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClientForBackendUnknownDefaultsToOllama(t *testing.T) {
	cfg := config.DefaultConfig()
	client, err := newClientForBackend(cfg, "unknown-backend")
	if err != nil {
		t.Fatalf("expected no error for unknown backend (defaults to ollama), got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClientForBackendOpenAINoKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	cfg := config.DefaultConfig()
	cfg.OpenAI.APIKey = ""
	_, err := newClientForBackend(cfg, "openai")
	if err == nil {
		t.Fatal("expected error when OpenAI API key is not set and env var is empty")
	}
}

func TestNewClientForBackendOpenAIWithConfigKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	cfg := config.DefaultConfig()
	cfg.OpenAI.APIKey = "sk-test"
	client, err := newClientForBackend(cfg, "openai")
	if err != nil {
		t.Fatalf("expected no error with API key set in config, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClientForBackendOpenAIFromEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-from-env")
	cfg := config.DefaultConfig() // APIKey is ""
	client, err := newClientForBackend(cfg, "openai")
	if err != nil {
		t.Fatalf("expected client when OPENAI_API_KEY is set, got error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}
