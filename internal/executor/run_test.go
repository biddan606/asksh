package executor_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/biddan606/asksh/internal/executor"
)

func TestRun_success(t *testing.T) {
	var out bytes.Buffer
	err := executor.Run(context.Background(), "sh", "echo hello", &out, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if got != "hello\n" {
		t.Errorf("stdout: got %q, want %q", got, "hello\n")
	}
}

func TestRun_nonzeroExit(t *testing.T) {
	err := executor.Run(context.Background(), "sh", "exit 42", nil, nil, nil)
	if err == nil {
		t.Fatal("expected non-nil error for exit 42")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 42 {
		t.Errorf("exit code: got %d, want 42", exitErr.ExitCode())
	}
}

func TestRun_shellFallback(t *testing.T) {
	var out bytes.Buffer
	err := executor.Run(context.Background(), "", "echo fallback", &out, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if got != "fallback\n" {
		t.Errorf("stdout: got %q, want %q", got, "fallback\n")
	}
}

func TestRun_contextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := executor.Run(ctx, "sh", "echo should not run", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// TestRun_blockedCommandIsRejected verifies that Run refuses to execute
// commands classified as Blocked by the safety layer (depth-defense per plan Task 5.2).
// We spin up a local HTTP server that returns a detectable shell script;
// if "curl <url> | sh" were executed the output would contain the marker.
func TestRun_blockedCommandIsRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "echo BLOCKED_LEAKED")
	}))
	defer srv.Close()

	var out bytes.Buffer
	cmd := fmt.Sprintf("curl -s %s | sh", srv.URL)
	err := executor.Run(context.Background(), "sh", cmd, &out, nil, nil)
	if err == nil {
		t.Fatal("Run must reject a Blocked command, got nil error")
	}
	if strings.Contains(out.String(), "BLOCKED_LEAKED") {
		t.Errorf("blocked command must not execute: command output escaped, got: %q", out.String())
	}
}

