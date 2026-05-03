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

func openaiServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": content}},
			},
		})
	}))
}

func TestOpenAITranslate(t *testing.T) {
	srv := openaiServer(t, "ls -la")
	defer srv.Close()

	c := llm.NewOpenAIClient("test-key", "gpt-4o-mini", srv.URL)
	sc := shellctx.ShellContext{CWD: "/tmp", OS: "darwin", Shell: "zsh"}

	got, err := c.Translate(context.Background(), "list all files", sc)
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if got != "ls -la" {
		t.Errorf("got %q; want %q", got, "ls -la")
	}
}

func TestOpenAISafetyCheck(t *testing.T) {
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
			srv := openaiServer(t, tc.content)
			defer srv.Close()

			c := llm.NewOpenAIClient("test-key", "gpt-4o-mini", srv.URL)
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

func TestOpenAIHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := llm.NewOpenAIClient("test-key", "gpt-4o-mini", srv.URL)
	sc := shellctx.ShellContext{}
	if _, err := c.Translate(context.Background(), "x", sc); err == nil {
		t.Fatal("expected error on HTTP 500")
	}
}

func TestOpenAIRequestFormat(t *testing.T) {
	var gotReq map[string]any
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotReq) //nolint:errcheck
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "echo hello"}},
			},
		})
	}))
	defer srv.Close()

	c := llm.NewOpenAIClient("sk-test", "gpt-4o-mini", srv.URL)
	sc := shellctx.ShellContext{}
	c.Translate(context.Background(), "say hello", sc) //nolint:errcheck

	if gotReq["model"] != "gpt-4o-mini" {
		t.Errorf("model = %v; want gpt-4o-mini", gotReq["model"])
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("Authorization = %q; want %q", gotAuth, "Bearer sk-test")
	}
}
