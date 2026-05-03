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
	cfg := config.DefaultConfig()
	cfg.OpenAI.APIKey = ""
	_, err := newClientForBackend(cfg, "openai")
	if err == nil {
		t.Fatal("expected error when OpenAI API key is not set")
	}
}

func TestNewClientForBackendOpenAIWithKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OpenAI.APIKey = "sk-test"
	client, err := newClientForBackend(cfg, "openai")
	if err != nil {
		t.Fatalf("expected no error with API key set, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}
