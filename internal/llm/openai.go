package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
		return "", fmt.Errorf("openai: HTTP %d", resp.StatusCode)
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
	lang := DetectLang(query)
	var buf bytes.Buffer
	if err := TranslateTemplate().Execute(&buf, TranslateData{
		CWD:   sc.CWD,
		OS:    sc.OS,
		Shell: sc.Shell,
		Lang:  lang,
		Query: query,
	}); err != nil {
		return "", err
	}
	return c.chat(ctx, buf.String())
}

func (c *OpenAIClient) SafetyCheck(ctx context.Context, cmd string) (Verdict, string, error) {
	var buf bytes.Buffer
	if err := SafetyTemplate().Execute(&buf, SafetyData{Cmd: cmd}); err != nil {
		return "", "", err
	}
	raw, err := c.chat(ctx, buf.String())
	if err != nil {
		return "", "", err
	}
	return parseSafety(raw)
}

func (c *OpenAIClient) Explain(ctx context.Context, cmd, lang string) (string, error) {
	var prompt string
	if lang == "ko" {
		prompt = "다음 쉘 명령을 간단히 설명해 주세요:\n" + cmd
	} else {
		prompt = "Briefly explain this shell command:\n" + cmd
	}
	return c.chat(ctx, prompt)
}
