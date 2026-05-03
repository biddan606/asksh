package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	shellctx "github.com/biddan606/asksh/internal/context"
	"github.com/biddan606/asksh/internal/llm"
)

func ollamaServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/chat" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"message": map[string]string{"role": "assistant", "content": content},
		})
	}))
}

func TestOllamaTranslate(t *testing.T) {
	srv := ollamaServer(t, "ls -la")
	defer srv.Close()

	c := llm.NewOllamaClient(srv.URL, "test-model")
	sc := shellctx.ShellContext{CWD: "/tmp", OS: "darwin", Shell: "zsh"}

	got, err := c.Translate(context.Background(), "list all files", sc)
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if got != "ls -la" {
		t.Errorf("got %q; want %q", got, "ls -la")
	}
}

func TestOllamaTranslateCleanup(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"plain", "ls -la", "ls -la"},
		{"leading dollar space", "$ ls -la", "ls -la"},
		{"code fence no lang", "```\nls -la\n```", "ls -la"},
		{"code fence with lang", "```bash\nls -la\n```", "ls -la"},
		{"code fence comment first line", "```bash\n# shows all files\nls -la\n```", "ls -la"},
		{"trailing newline", "ls -la\n", "ls -la"},
		{"trailing spaces", "ls -la   ", "ls -la"},
		{"command prefix", "Command: ls -la", "ls -la"},
		{"korean prefix", "명령: ls -la", "ls -la"},
		{"multiline take first", "ls -la\nThis lists all files.", "ls -la"},
		{"multiline empty first line", "\nls -la\n", "ls -la"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := ollamaServer(t, tc.raw)
			defer srv.Close()

			c := llm.NewOllamaClient(srv.URL, "test-model")
			sc := shellctx.ShellContext{}

			got, err := c.Translate(context.Background(), "x", sc)
			if err != nil {
				t.Fatalf("Translate: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}

func TestOllamaSafetyCheck(t *testing.T) {
	cases := []struct {
		name    string
		content string
		verdict llm.Verdict
	}{
		{"safe", "safe looks fine", llm.VerdictSafe},
		{"warn", "warn risky operation", llm.VerdictWarn},
		{"dangerous", "dangerous deletes files", llm.VerdictDangerous},
		{"unknown fallback to warn", "what is this", llm.VerdictWarn},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := ollamaServer(t, tc.content)
			defer srv.Close()

			c := llm.NewOllamaClient(srv.URL, "test-model")
			v, _, err := c.SafetyCheck(context.Background(), "ls")
			if err != nil {
				t.Fatalf("SafetyCheck: %v", err)
			}
			if v != tc.verdict {
				t.Errorf("verdict = %q; want %q", v, tc.verdict)
			}
		})
	}
}

func TestOllamaHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := llm.NewOllamaClient(srv.URL, "test-model")
	sc := shellctx.ShellContext{}
	if _, err := c.Translate(context.Background(), "x", sc); err == nil {
		t.Fatal("expected error on HTTP 500")
	}
}

func TestOllamaRequestFormat(t *testing.T) {
	var gotReq map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotReq) //nolint:errcheck
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"message": map[string]string{"role": "assistant", "content": "echo hello"},
		})
	}))
	defer srv.Close()

	c := llm.NewOllamaClient(srv.URL, "my-model")
	sc := shellctx.ShellContext{}
	c.Translate(context.Background(), "say hello", sc) //nolint:errcheck

	if gotReq["model"] != "my-model" {
		t.Errorf("model = %v; want my-model", gotReq["model"])
	}
	if gotReq["stream"] != false {
		t.Errorf("stream = %v; want false", gotReq["stream"])
	}
}

func TestOllamaExplain(t *testing.T) {
	srv := ollamaServer(t, "Lists files including hidden ones.")
	defer srv.Close()

	c := llm.NewOllamaClient(srv.URL, "test-model")
	got, err := c.Explain(context.Background(), "ls -la", "en")
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if got != "Lists files including hidden ones." {
		t.Errorf("got %q; want explanation text", got)
	}
}

func TestOllamaExplainKorean(t *testing.T) {
	var gotPrompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		if msgs, ok := req["messages"].([]any); ok && len(msgs) > 0 {
			gotPrompt, _ = msgs[0].(map[string]any)["content"].(string)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"message": map[string]string{"role": "assistant", "content": "모든 파일을 나열합니다."},
		})
	}))
	defer srv.Close()

	c := llm.NewOllamaClient(srv.URL, "test-model")
	if _, err := c.Explain(context.Background(), "ls -la", "ko"); err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if !strings.Contains(gotPrompt, "다음 쉘 명령") {
		t.Errorf("Korean Explain prompt missing Korean text, got: %q", gotPrompt)
	}
}

func TestOllamaHTTPErrorIncludesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid credentials"}`)) //nolint:errcheck
	}))
	defer srv.Close()

	c := llm.NewOllamaClient(srv.URL, "test-model")
	sc := shellctx.ShellContext{}
	_, err := c.Translate(context.Background(), "x", sc)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("error missing response body, got: %v", err)
	}
}

func TestOllamaJSONDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json")) //nolint:errcheck
	}))
	defer srv.Close()

	c := llm.NewOllamaClient(srv.URL, "test-model")
	sc := shellctx.ShellContext{}
	if _, err := c.Translate(context.Background(), "x", sc); err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestOllamaSafetyCheckReason(t *testing.T) {
	srv := ollamaServer(t, "warn modifies system files")
	defer srv.Close()

	c := llm.NewOllamaClient(srv.URL, "test-model")
	verdict, reason, err := c.SafetyCheck(context.Background(), "chmod 777 /etc/passwd")
	if err != nil {
		t.Fatalf("SafetyCheck: %v", err)
	}
	if verdict != llm.VerdictWarn {
		t.Errorf("verdict = %q; want warn", verdict)
	}
	if !strings.Contains(reason, "modifies system files") {
		t.Errorf("reason = %q; want to contain 'modifies system files'", reason)
	}
}
