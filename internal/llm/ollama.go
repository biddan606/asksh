package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	shellctx "github.com/biddan606/asksh/internal/context"
)

type OllamaClient struct {
	host   string
	model  string
	client *http.Client
}

func NewOllamaClient(host, model string) *OllamaClient {
	return &OllamaClient{
		host:   host,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}

func (c *OllamaClient) chat(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(ollamaRequest{
		Model:    c.model,
		Messages: []ollamaMessage{{Role: "user", Content: prompt}},
		Stream:   false,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("ollama: HTTP %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return cleanCmd(result.Message.Content), nil
}

func (c *OllamaClient) Translate(ctx context.Context, query string, sc shellctx.ShellContext) (string, error) {
	return translate(c.chat, ctx, query, sc)
}

func (c *OllamaClient) SafetyCheck(ctx context.Context, cmd string) (Verdict, string, error) {
	return checkSafety(c.chat, ctx, cmd)
}

func (c *OllamaClient) Explain(ctx context.Context, cmd, lang string) (string, error) {
	return explain(c.chat, ctx, cmd, lang)
}
