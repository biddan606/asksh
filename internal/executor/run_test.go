package executor_test

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/biddan606/asksh/internal/executor"
)

func TestRun_success(t *testing.T) {
	var out bytes.Buffer
	err := executor.Run(context.Background(), "sh", "echo hello", &out, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if got != "hello\n" {
		t.Errorf("stdout: got %q, want %q", got, "hello\n")
	}
}

func TestRun_nonzeroExit(t *testing.T) {
	err := executor.Run(context.Background(), "sh", "exit 42", nil, nil)
	if err == nil {
		t.Fatal("expected non-nil error for exit 42")
	}
	var exitErr *exec.ExitError
	if ok := isExitError(err, &exitErr); !ok {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 42 {
		t.Errorf("exit code: got %d, want 42", exitErr.ExitCode())
	}
}

func TestRun_shellFallback(t *testing.T) {
	var out bytes.Buffer
	err := executor.Run(context.Background(), "", "echo fallback", &out, nil)
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
	err := executor.Run(ctx, "sh", "echo should not run", nil, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// isExitError is a helper so we don't import errors in test.
func isExitError(err error, target **exec.ExitError) bool {
	ee, ok := err.(*exec.ExitError)
	if ok {
		*target = ee
	}
	return ok
}
