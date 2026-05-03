package prompt_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	shellctx "github.com/biddan606/asksh/internal/context"
	"github.com/biddan606/asksh/internal/llm"
	"github.com/biddan606/asksh/internal/prompt"
)

type mockClient struct{ explanation string }

func (m *mockClient) Translate(_ context.Context, _ string, _ shellctx.ShellContext) (string, error) {
	return "", nil
}
func (m *mockClient) SafetyCheck(_ context.Context, _ string) (llm.Verdict, string, error) {
	return llm.VerdictSafe, "", nil
}
func (m *mockClient) Explain(_ context.Context, _, _ string) (string, error) {
	return m.explanation, nil
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
