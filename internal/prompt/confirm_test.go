package prompt_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	shellctx "github.com/biddan606/asksh/internal/context"
	"github.com/biddan606/asksh/internal/llm"
	"github.com/biddan606/asksh/internal/prompt"
)

type mockClient struct {
	explanation string
	explainErr  error
}

func (m *mockClient) Translate(_ context.Context, _ string, _ shellctx.ShellContext) (string, error) {
	return "", nil
}
func (m *mockClient) SafetyCheck(_ context.Context, _ string) (llm.Verdict, string, error) {
	return llm.VerdictSafe, "", nil
}
func (m *mockClient) Explain(_ context.Context, _, _ string) (string, error) {
	return m.explanation, m.explainErr
}

func TestAsk(t *testing.T) {
	const origCmd = "rm -rf ./node_modules"
	const editedCmd = "rm -rf ./dist"
	client := &mockClient{explanation: "이 명령은 node_modules 디렉토리를 삭제합니다."}

	tests := []struct {
		name       string
		input      string
		wantAction prompt.Action
		wantCmd    string
	}{
		{
			name:       "y executes original command",
			input:      "y\n",
			wantAction: prompt.Execute,
			wantCmd:    origCmd,
		},
		{
			name:       "n cancels",
			input:      "n\n",
			wantAction: prompt.Cancel,
			wantCmd:    "",
		},
		{
			name:       "e edit then y executes edited command",
			input:      "e\n" + editedCmd + "\ny\n",
			wantAction: prompt.Execute,
			wantCmd:    editedCmd,
		},
		{
			name:       "? explain then y executes",
			input:      "?\ny\n",
			wantAction: prompt.Execute,
			wantCmd:    origCmd,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			p := prompt.Prompt{
				Ctx:     context.Background(),
				Cmd:     origCmd,
				Warning: "이 명령은 파괴적이며 되돌릴 수 없습니다.",
				Reason:  "디렉토리 트리를 재귀적으로 삭제합니다",
				Client:  client,
				Lang:    "ko",
				Out:     &out,
				In:      strings.NewReader(tc.input),
			}
			action, cmd, err := prompt.Ask(p)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if action != tc.wantAction {
				t.Errorf("action: got %v, want %v", action, tc.wantAction)
			}
			if cmd != tc.wantCmd {
				t.Errorf("cmd: got %q, want %q", cmd, tc.wantCmd)
			}
		})
	}
}

func TestAsk_nilCtxUsesBackground(t *testing.T) {
	var out bytes.Buffer
	p := prompt.Prompt{
		Ctx:    nil, // must not panic
		Cmd:    "echo hi",
		Client: &mockClient{},
		Out:    &out,
		In:     strings.NewReader("n\n"),
	}
	action, _, err := prompt.Ask(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != prompt.Cancel {
		t.Errorf("expected Cancel, got %v", action)
	}
}

func TestAsk_eofReturnsCancelClean(t *testing.T) {
	var out bytes.Buffer
	p := prompt.Prompt{
		Ctx:    context.Background(),
		Cmd:    "echo hi",
		Client: &mockClient{},
		Out:    &out,
		In:     strings.NewReader(""), // immediate EOF
	}
	action, _, err := prompt.Ask(p)
	if err != nil {
		t.Fatalf("unexpected error on EOF: %v", err)
	}
	if action != prompt.Cancel {
		t.Errorf("expected Cancel on EOF, got %v", action)
	}
}

func TestAsk_emptyEditContinues(t *testing.T) {
	// "e" then empty line → loop repeats; "y" on third prompt executes.
	var out bytes.Buffer
	p := prompt.Prompt{
		Ctx:    context.Background(),
		Cmd:    "echo original",
		Client: &mockClient{},
		Out:    &out,
		In:     strings.NewReader("e\n\ny\n"),
	}
	action, cmd, err := prompt.Ask(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != prompt.Execute {
		t.Errorf("expected Execute after empty-edit+y, got %v", action)
	}
	if cmd != "echo original" {
		t.Errorf("cmd should remain original after empty edit, got %q", cmd)
	}
}

func TestAsk_editToBlockedCancels(t *testing.T) {
	var out bytes.Buffer
	p := prompt.Prompt{
		Ctx:    context.Background(),
		Cmd:    "rm -rf ./build",
		Client: &mockClient{},
		Out:    &out,
		In:     strings.NewReader("e\nsudo rm -rf /\n"),
	}
	action, _, err := prompt.Ask(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != prompt.Cancel {
		t.Errorf("expected Cancel when editing to blocked cmd, got %v", action)
	}
	if !strings.Contains(out.String(), "차단됨") {
		t.Errorf("expected '차단됨' in output, got: %q", out.String())
	}
}

func TestAsk_editToDangerousUpdatesWarning(t *testing.T) {
	var out bytes.Buffer
	p := prompt.Prompt{
		Ctx:     context.Background(),
		Cmd:     "echo safe",
		Warning: "",
		Reason:  "",
		Client:  &mockClient{},
		Out:     &out,
		In:      strings.NewReader("e\nrm -rf ./dist\ny\n"),
	}
	action, cmd, err := prompt.Ask(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != prompt.Execute {
		t.Errorf("expected Execute, got %v", action)
	}
	if cmd != "rm -rf ./dist" {
		t.Errorf("expected edited cmd, got %q", cmd)
	}
	if !strings.Contains(out.String(), "⚠") {
		t.Errorf("expected warning after dangerous edit, got: %q", out.String())
	}
}

func TestAsk_editToSafeClearsWarning(t *testing.T) {
	var out bytes.Buffer
	p := prompt.Prompt{
		Ctx:     context.Background(),
		Cmd:     "rm -rf ./build",
		Warning: "이 명령은 파괴적이며 되돌릴 수 없습니다.",
		Reason:  "recursive rm is dangerous",
		Client:  &mockClient{},
		Out:     &out,
		In:      strings.NewReader("e\necho safe_cmd\nn\n"),
	}
	action, _, err := prompt.Ask(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != prompt.Cancel {
		t.Errorf("expected Cancel, got %v", action)
	}
	// After edit to safe command the re-render should not show the warning.
	// The warning appears before each prompt; the last render (after edit) should lack ⚠.
	outStr := out.String()
	// Split by render cycles — the second render block should have no ⚠.
	parts := strings.SplitN(outStr, "번역된 명령어:", 3)
	if len(parts) >= 3 && strings.Contains(parts[2], "⚠") {
		t.Errorf("warning should be cleared after editing to safe command, got: %q", parts[2])
	}
}

func TestAsk_nilClientExplainShowsMessage(t *testing.T) {
	var out bytes.Buffer
	p := prompt.Prompt{
		Ctx:    context.Background(),
		Cmd:    "echo hi",
		Client: nil,
		Out:    &out,
		In:     strings.NewReader("?\ny\n"),
	}
	action, _, err := prompt.Ask(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != prompt.Execute {
		t.Errorf("expected Execute after nil-client explain, got %v", action)
	}
	if !strings.Contains(out.String(), "LLM 클라이언트") {
		t.Errorf("expected nil-client message in output, got: %q", out.String())
	}
}

func TestAsk_noColorEnvNoANSI(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var out bytes.Buffer
	p := prompt.Prompt{
		Ctx:     context.Background(),
		Cmd:     "echo hello",
		Warning: "dangerous warning",
		Client:  &mockClient{},
		Out:     &out,
		In:      strings.NewReader("n\n"),
	}
	prompt.Ask(p)
	if strings.Contains(out.String(), "\033[") {
		t.Error("ANSI escape codes must not appear when NO_COLOR is set")
	}
}

func TestAsk_explainErrorShowsError(t *testing.T) {
	var out bytes.Buffer
	p := prompt.Prompt{
		Ctx:    context.Background(),
		Cmd:    "echo hi",
		Client: &mockClient{explainErr: errors.New("connection refused")},
		Out:    &out,
		In:     strings.NewReader("?\ny\n"),
	}
	action, _, err := prompt.Ask(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != prompt.Execute {
		t.Errorf("expected Execute after explain error, got %v", action)
	}
	if !strings.Contains(out.String(), "설명 오류") {
		t.Errorf("expected '설명 오류' in output, got: %q", out.String())
	}
}
