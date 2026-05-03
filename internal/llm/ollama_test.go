package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
