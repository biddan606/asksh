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
