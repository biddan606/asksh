package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biddan606/asksh/cmd"
	"github.com/biddan606/asksh/internal/config"
	shellctx "github.com/biddan606/asksh/internal/context"
	"github.com/biddan606/asksh/internal/history"
	"github.com/biddan606/asksh/internal/llm"
)

type mockClient struct {
	result        string
	err           error
	safetyVerdict llm.Verdict
	safetyReason  string
	safetyErr     error
}

func (m *mockClient) Translate(_ context.Context, _ string, _ shellctx.ShellContext) (string, error) {
	return m.result, m.err
}

func (m *mockClient) SafetyCheck(_ context.Context, _ string) (llm.Verdict, string, error) {
	return m.safetyVerdict, m.safetyReason, m.safetyErr
}

func (m *mockClient) Explain(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func executeCommandWithFactory(cfg config.Config, factory func(string) (llm.Client, error), args ...string) (string, error) {
	return executeCommandWithInput("", cfg, factory, args...)
}

func executeCommandWithInput(stdin string, cfg config.Config, factory func(string) (llm.Client, error), args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd(cfg, factory)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func executeCommand(args ...string) (string, error) {
	return executeCommandWithFactory(config.DefaultConfig(), nil, args...)
}

func executeCommandWithConfig(cfg config.Config, args ...string) (string, error) {
	return executeCommandWithFactory(cfg, nil, args...)
}

func TestVersion(t *testing.T) {
	out, err := executeCommand("--version")
	if err != nil {
		t.Fatalf("--version returned error: %v", err)
	}
	if !strings.Contains(out, "0.1.0-dev") {
		t.Errorf("--version output %q does not contain 0.1.0-dev", out)
	}
}

func TestHelp(t *testing.T) {
	_, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
}

func TestDryRun(t *testing.T) {
	out, err := executeCommand("--dry-run", "list files")
	if err != nil {
		t.Fatalf("--dry-run returned error: %v", err)
	}
	if !strings.Contains(out, "list files") {
		t.Errorf("--dry-run output %q does not contain query", out)
	}
}

func TestDryRunJoinsArgs(t *testing.T) {
	out, err := executeCommand("--dry-run", "list", "all", "files")
	if err != nil {
		t.Fatalf("--dry-run returned error: %v", err)
	}
	if !strings.Contains(out, "list all files") {
		t.Errorf("expected joined args in output, got: %q", out)
	}
}

func TestBareCommandErrors(t *testing.T) {
	_, err := executeCommand()
	if err == nil {
		t.Error("bare asksh (no args) should return an error")
	}
}

func TestBackendFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"ollama alone", []string{"--ollama", "--dry-run", "x"}},
		{"openai alone", []string{"--openai", "--dry-run", "x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if err != nil {
				t.Errorf("args %v returned error: %v", tt.args, err)
			}
		})
	}
}

func TestMutuallyExclusiveBackendFlags(t *testing.T) {
	_, err := executeCommand("--ollama", "--openai", "--dry-run", "x")
	if err == nil {
		t.Error("--ollama and --openai together should return an error")
	}
}

func TestHelpContainsUsage(t *testing.T) {
	out, _ := executeCommand("--help")
	if !strings.Contains(out, "asksh") {
		t.Errorf("--help output %q does not contain 'asksh'", out)
	}
}

func TestDryRunShowsShellContext(t *testing.T) {
	out, err := executeCommand("--dry-run", "x")
	if err != nil {
		t.Fatalf("--dry-run returned error: %v", err)
	}
	for _, key := range []string{"cwd=", "os=", "shell="} {
		if !strings.Contains(out, key) {
			t.Errorf("--dry-run output %q does not contain %q", out, key)
		}
	}
}

// Checkpoint A: verify exact CLI flag names and dry-run output format.

func TestDryRunQueryLineFormat(t *testing.T) {
	out, err := executeCommand("--dry-run", "list files")
	if err != nil {
		t.Fatalf("--dry-run returned error: %v", err)
	}
	if !strings.Contains(out, "query: list files") {
		t.Errorf("expected 'query: list files' in output, got: %q", out)
	}
}

func TestDryRunContextOnSameLine(t *testing.T) {
	out, err := executeCommand("--dry-run", "x")
	if err != nil {
		t.Fatalf("--dry-run returned error: %v", err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "cwd=") {
			if !strings.Contains(line, "os=") || !strings.Contains(line, "shell=") {
				t.Errorf("cwd=, os=, shell= must be on same line; got: %q", line)
			}
			return
		}
	}
	t.Error("no line containing cwd= found in output")
}

func TestDryRunDoesNotExecute(t *testing.T) {
	out, err := executeCommand("--dry-run", "x")
	if err != nil {
		t.Fatalf("--dry-run returned error: %v", err)
	}
	if strings.Contains(out, "not yet implemented") {
		t.Errorf("--dry-run must not show execution placeholder, got: %q", out)
	}
}

func TestConfigSubcommandRegistered(t *testing.T) {
	_, err := executeCommand("config", "--help")
	if err != nil {
		t.Fatalf("config subcommand should be registered, got: %v", err)
	}
}

func TestDryRunFlagWithoutQueryErrors(t *testing.T) {
	_, err := executeCommand("--dry-run")
	if err == nil {
		t.Error("--dry-run without query should return an error")
	}
}

func TestDryRunQueryBeforeContext(t *testing.T) {
	out, err := executeCommand("--dry-run", "x")
	if err != nil {
		t.Fatalf("--dry-run returned error: %v", err)
	}
	queryIdx := strings.Index(out, "query:")
	cwdIdx := strings.Index(out, "cwd=")
	if queryIdx == -1 || cwdIdx == -1 {
		t.Fatalf("output missing query: or cwd=, got: %q", out)
	}
	if queryIdx >= cwdIdx {
		t.Errorf("query: line should appear before cwd= line, got: %q", out)
	}
}

func TestDryRunShowsBackend(t *testing.T) {
	out, err := executeCommand("--dry-run", "x")
	if err != nil {
		t.Fatalf("--dry-run returned error: %v", err)
	}
	if !strings.Contains(out, "backend=ollama") {
		t.Errorf("expected backend=ollama in output, got: %q", out)
	}
}

func TestDryRunOllamaFlagOverridesConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Backend.Default = "openai"
	out, err := executeCommandWithConfig(cfg, "--ollama", "--dry-run", "x")
	if err != nil {
		t.Fatalf("--ollama returned error: %v", err)
	}
	if !strings.Contains(out, "backend=ollama") {
		t.Errorf("--ollama flag should override config to ollama, got: %q", out)
	}
}

func TestDryRunOpenAIFlagOverridesConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Backend.Default = "ollama"
	out, err := executeCommandWithConfig(cfg, "--openai", "--dry-run", "x")
	if err != nil {
		t.Fatalf("--openai returned error: %v", err)
	}
	if !strings.Contains(out, "backend=openai") {
		t.Errorf("--openai flag should override config to openai, got: %q", out)
	}
}

func TestDryRunUsesConfigBackend(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Backend.Default = "openai"
	out, err := executeCommandWithConfig(cfg, "--dry-run", "x")
	if err != nil {
		t.Fatalf("--dry-run returned error: %v", err)
	}
	if !strings.Contains(out, "backend=openai") {
		t.Errorf("config backend=openai should be used when no flag set, got: %q", out)
	}
}

func TestTranslateOutputsCommand(t *testing.T) {
	mock := &mockClient{result: "ls -la"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.ExtraLLMCheck = false
	out, err := executeCommandWithInput("n\n", cfg, factory, "list files")
	if err != nil && !errors.Is(err, cmd.ErrCancelled) {
		t.Fatalf("translate returned unexpected error: %v", err)
	}
	if !strings.Contains(out, "ls -la") {
		t.Errorf("expected 'ls -la' in output, got: %q", out)
	}
}

func TestTranslateErrorPropagates(t *testing.T) {
	mock := &mockClient{err: errors.New("LLM unavailable")}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	_, err := executeCommandWithFactory(config.DefaultConfig(), factory, "list files")
	if err == nil {
		t.Error("expected error when LLM fails")
	}
}

func TestTranslateNoClientErrors(t *testing.T) {
	_, err := executeCommandWithFactory(config.DefaultConfig(), nil, "list files")
	if err == nil {
		t.Error("expected error when no client factory provided")
	}
}

func TestDryRunSafetyBlockedReturnsError(t *testing.T) {
	_, err := executeCommand("--dry-run", "rm -rf /")
	if err == nil {
		t.Error("BLOCKED command should return error in dry-run")
	}
}

func TestDryRunSafetyBlockedOutputContainsBlocked(t *testing.T) {
	out, _ := executeCommand("--dry-run", "rm -rf /")
	if !strings.Contains(strings.ToUpper(out), "BLOCKED") {
		t.Errorf("BLOCKED command output should contain BLOCKED, got: %q", out)
	}
}

func TestDryRunSafetySafeOutput(t *testing.T) {
	out, err := executeCommand("--dry-run", "ls")
	if err != nil {
		t.Fatalf("unexpected error for safe command: %v", err)
	}
	if !strings.Contains(strings.ToUpper(out), "SAFE") {
		t.Errorf("safe command dry-run output should contain SAFE, got: %q", out)
	}
}

func TestDryRunSafetyDangerousOutput(t *testing.T) {
	out, err := executeCommand("--dry-run", "rm -rf ./node_modules")
	if err != nil {
		t.Fatalf("unexpected error for dangerous (non-blocked) command: %v", err)
	}
	upper := strings.ToUpper(out)
	if !strings.Contains(upper, "DANGEROUS") {
		t.Errorf("dangerous command dry-run output should contain DANGEROUS, got: %q", out)
	}
	if !strings.Contains(strings.ToLower(out), "rule") {
		t.Errorf("dangerous rule output should mention 'rule', got: %q", out)
	}
}

func TestDryRunSafetyLLMVerdictShown(t *testing.T) {
	mock := &mockClient{safetyVerdict: llm.VerdictDangerous, safetyReason: "deletes files"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	out, _ := executeCommandWithFactory(config.DefaultConfig(), factory, "--dry-run", "rm -rf ./node_modules")
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "llm") {
		t.Errorf("dry-run with client should show llm verdict, got: %q", out)
	}
	if !strings.Contains(lower, "dangerous") {
		t.Errorf("dry-run with dangerous llm verdict should show dangerous, got: %q", out)
	}
}

func TestDryRunLLMErrorShown(t *testing.T) {
	mock := &mockClient{safetyErr: errors.New("connection refused")}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	out, _ := executeCommandWithFactory(config.DefaultConfig(), factory, "--dry-run", "ls")
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "llm") {
		t.Errorf("dry-run should show llm line even on error, got: %q", out)
	}
	if !strings.Contains(lower, "error") && !strings.Contains(lower, "unavailable") {
		t.Errorf("dry-run should indicate llm error, got: %q", out)
	}
}

// Task 5.3: full pipeline tests (translate → safety → prompt → execute)

func TestPipelineBlockedTranslatedErrors(t *testing.T) {
	mock := &mockClient{result: "rm -rf /"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.ExtraLLMCheck = false
	_, err := executeCommandWithInput("", cfg, factory, "delete everything")
	if err == nil {
		t.Error("expected error when translated command is blocked")
	}
}

func TestPipelineUserCancelExitsNonZero(t *testing.T) {
	mock := &mockClient{result: "echo hello"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.ExtraLLMCheck = false
	_, err := executeCommandWithInput("n\n", cfg, factory, "say hello")
	if !errors.Is(err, cmd.ErrCancelled) {
		t.Errorf("cancel should return ErrCancelled, got: %v", err)
	}
}

func TestPipelineExecutesOnConfirm(t *testing.T) {
	mock := &mockClient{result: "echo asksh_test_marker"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.ExtraLLMCheck = false
	out, err := executeCommandWithInput("y\n", cfg, factory, "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "asksh_test_marker") {
		t.Errorf("expected command output after confirm, got: %q", out)
	}
}

func TestPipelineDangerousShowsWarning(t *testing.T) {
	mock := &mockClient{result: "rm -rf ./build"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.ExtraLLMCheck = false
	out, err := executeCommandWithInput("n\n", cfg, factory, "clean build dir")
	if err != nil && !errors.Is(err, cmd.ErrCancelled) {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "⚠") {
		t.Errorf("expected warning symbol in output, got: %q", out)
	}
}

func TestPipelineNoConfirmExecutes(t *testing.T) {
	mock := &mockClient{result: "echo asksh_noconfirm"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.RequireConfirmation = false
	cfg.Safety.ExtraLLMCheck = false
	out, err := executeCommandWithInput("", cfg, factory, "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "asksh_noconfirm") {
		t.Errorf("expected command executed without confirmation, got: %q", out)
	}
}

func TestPipelineExtraLLMCheckApplied(t *testing.T) {
	mock := &mockClient{
		result:        "ls -la",
		safetyVerdict: llm.VerdictDangerous,
		safetyReason:  "potential data exposure",
	}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.ExtraLLMCheck = true
	// SPEC §2.3: dangerous → 차단 후 이유 설명
	_, err := executeCommandWithInput("", cfg, factory, "list files")
	if err == nil {
		t.Fatal("expected error for LLM dangerous verdict, got nil")
	}
	if !strings.Contains(err.Error(), "BLOCKED (LLM)") {
		t.Errorf("expected BLOCKED (LLM) in error, got: %v", err)
	}
}

// Checkpoint E: end-to-end usability session

func TestPipelineSafeRequireConfirmExecutes(t *testing.T) {
	mock := &mockClient{result: "echo e2e_confirm"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.RequireConfirmation = true
	cfg.Safety.ExtraLLMCheck = false
	out, err := executeCommandWithInput("y\n", cfg, factory, "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "e2e_confirm") {
		t.Errorf("expected command output after confirm, got: %q", out)
	}
}

func TestPipelineSafeRequireConfirmCancel(t *testing.T) {
	mock := &mockClient{result: "echo e2e_cancel"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.RequireConfirmation = true
	cfg.Safety.ExtraLLMCheck = false
	// cancel should return ErrCancelled (exit code 1); command must not execute
	_, err := executeCommandWithInput("n\n", cfg, factory, "say hello")
	if !errors.Is(err, cmd.ErrCancelled) {
		t.Fatalf("cancel should return ErrCancelled, got: %v", err)
	}
}

func TestPipelineSafeRequireConfirmShowsNoWarning(t *testing.T) {
	mock := &mockClient{result: "echo safe_cmd"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.RequireConfirmation = true
	cfg.Safety.ExtraLLMCheck = false
	out, err := executeCommandWithInput("n\n", cfg, factory, "safe query")
	if err != nil && !errors.Is(err, cmd.ErrCancelled) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "⚠") {
		t.Errorf("safe command should not show warning, got: %q", out)
	}
	if !strings.Contains(out, "echo safe_cmd") {
		t.Errorf("prompt should show translated command, got: %q", out)
	}
}

func TestPipelineEditThenExecute(t *testing.T) {
	mock := &mockClient{result: "rm -rf ./build"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.ExtraLLMCheck = false
	// edit dangerous command to safe, then confirm
	out, err := executeCommandWithInput("e\necho edited_output\ny\n", cfg, factory, "clean build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "edited_output") {
		t.Errorf("expected edited command output, got: %q", out)
	}
}

func TestPipelineEditToBlockedCancels(t *testing.T) {
	mock := &mockClient{result: "rm -rf ./build"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.Safety.ExtraLLMCheck = false
	_, err := executeCommandWithInput("e\nrm -rf /\n", cfg, factory, "clean build")
	if !errors.Is(err, cmd.ErrCancelled) {
		t.Fatalf("edit to blocked command should return ErrCancelled, got: %v", err)
	}
}

// Task 6.2: exit codes, NO_COLOR, help text

func TestErrorDoesNotShowUsage(t *testing.T) {
	// When a command fails, cobra must not dump the usage/help text.
	out, _ := executeCommandWithFactory(config.DefaultConfig(), nil, "list files")
	if strings.Contains(out, "Usage:") {
		t.Errorf("error output must not show Usage:, got: %q", out)
	}
}

func TestHelpHasLongDesc(t *testing.T) {
	out, _ := executeCommand("--help")
	// Long description should mention natural language capability
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "natural language") && !strings.Contains(lower, "자연어") {
		t.Errorf("--help should include a long description, got: %q", out)
	}
}

func TestHelpHasExamples(t *testing.T) {
	out, _ := executeCommand("--help")
	if !strings.Contains(strings.ToLower(out), "example") {
		t.Errorf("--help should have an Examples section, got: %q", out)
	}
}

// logHistory integration tests

func readHistoryEntries(t *testing.T, dir string) []history.Entry {
	t.Helper()
	path := filepath.Join(dir, "asksh", "history.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	var entries []history.Entry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var e history.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("unmarshal %q: %v", line, err)
		}
		entries = append(entries, e)
	}
	return entries
}

func TestPipelineHistoryWrittenOnExecute(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	mock := &mockClient{result: "echo history_test"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.History.Enable = true
	cfg.Safety.ExtraLLMCheck = false

	_, err := executeCommandWithInput("y\n", cfg, factory, "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := readHistoryEntries(t, dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Query != "say hello" {
		t.Errorf("entry.Query = %q, want %q", e.Query, "say hello")
	}
	if e.Command != "echo history_test" {
		t.Errorf("entry.Command = %q, want %q", e.Command, "echo history_test")
	}
	if e.Result != "ok" {
		t.Errorf("entry.Result = %q, want \"ok\"", e.Result)
	}
}

func TestPipelineHistoryWrittenOnCancel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	mock := &mockClient{result: "echo history_cancel"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.History.Enable = true
	cfg.Safety.ExtraLLMCheck = false

	_, err := executeCommandWithInput("n\n", cfg, factory, "cancel query")
	if !errors.Is(err, cmd.ErrCancelled) {
		t.Fatalf("expected ErrCancelled, got: %v", err)
	}

	entries := readHistoryEntries(t, dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(entries))
	}
	if entries[0].Result != "cancelled" {
		t.Errorf("entry.Result = %q, want \"cancelled\"", entries[0].Result)
	}
}

func TestPipelineHistoryNotWrittenWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	mock := &mockClient{result: "echo no_history"}
	factory := func(_ string) (llm.Client, error) { return mock, nil }
	cfg := config.DefaultConfig()
	cfg.History.Enable = false
	cfg.Safety.ExtraLLMCheck = false

	_, _ = executeCommandWithInput("y\n", cfg, factory, "say hello")

	path := filepath.Join(dir, "asksh", "history.log")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("history file should not exist when disabled, but stat returned: %v", err)
	}
}
