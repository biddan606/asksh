package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
		return "", fmt.Errorf("ollama: HTTP %d", resp.StatusCode)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return cleanCmd(result.Message.Content), nil
}

// cleanCmd removes code fences, known LLM prefixes, and normalises whitespace.
func cleanCmd(s string) string {
	s = strings.TrimSpace(s)
	// Extract from code fence if present.
	if i := strings.Index(s, "```"); i != -1 {
		s = s[i+3:]
		if nl := strings.Index(s, "\n"); nl != -1 {
			s = s[nl+1:]
		}
		if end := strings.Index(s, "```"); end != -1 {
			s = s[:end]
		}
		s = strings.TrimSpace(s)
	}
	// Strip prefixes the LLM may add despite instructions.
	for _, pfx := range []string{"Command: ", "명령: ", "$ "} {
		if after, ok := strings.CutPrefix(s, pfx); ok {
			s = after
			break
		}
	}
	// If the response contains multiple lines, take the first non-empty one.
	if i := strings.IndexByte(s, '\n'); i != -1 {
		first := strings.TrimSpace(s[:i])
		if first != "" {
			s = first
		} else {
			s = strings.TrimSpace(s[i+1:])
		}
	}
	return strings.TrimSpace(s)
}

func (c *OllamaClient) Translate(ctx context.Context, query string, sc shellctx.ShellContext) (string, error) {
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

func (c *OllamaClient) SafetyCheck(ctx context.Context, cmd string) (Verdict, string, error) {
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

func parseSafety(raw string) (Verdict, string, error) {
	raw = strings.TrimSpace(raw)
	word, reason, _ := strings.Cut(raw, " ")
	switch v := Verdict(strings.ToLower(word)); v {
	case VerdictSafe, VerdictWarn, VerdictDangerous:
		return v, reason, nil
	default:
		return VerdictWarn, raw, nil
	}
}

func (c *OllamaClient) Explain(ctx context.Context, cmd, lang string) (string, error) {
	var prompt string
	if lang == "ko" {
		prompt = "다음 쉘 명령을 간단히 설명해 주세요:\n" + cmd
	} else {
		prompt = "Briefly explain this shell command:\n" + cmd
	}
	return c.chat(ctx, prompt)
}
