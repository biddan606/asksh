package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/biddan606/asksh/cmd"
)

func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
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

func TestWithoutDryRunShowsPlaceholder(t *testing.T) {
	out, err := executeCommand("list files")
	if err != nil {
		t.Fatalf("non-dry-run returned error: %v", err)
	}
	if !strings.Contains(out, "not yet implemented") {
		t.Errorf("without --dry-run expected placeholder, got: %q", out)
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
