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

type OpenAIClient struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAIClient(apiKey, model, baseURL string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &OpenAIClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func (c *OpenAIClient) chat(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(openAIRequest{
		Model:    c.model,
		Messages: []openAIMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("openai: empty choices")
	}

	return cleanCmd(result.Choices[0].Message.Content), nil
}

func (c *OpenAIClient) Translate(ctx context.Context, query string, sc shellctx.ShellContext) (string, error) {
	return translate(c.chat, ctx, query, sc)
}

func (c *OpenAIClient) SafetyCheck(ctx context.Context, cmd string) (Verdict, string, error) {
	return checkSafety(c.chat, ctx, cmd)
}

func (c *OpenAIClient) Explain(ctx context.Context, cmd, lang string) (string, error) {
	return explain(c.chat, ctx, cmd, lang)
}
