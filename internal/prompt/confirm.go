package prompt

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/biddan606/asksh/internal/llm"
	"github.com/biddan606/asksh/internal/safety"
	"golang.org/x/term"
)

type Action int

const (
	Execute Action = iota
	Cancel
)

// Prompt holds parameters for the interactive confirmation dialog.
type Prompt struct {
	Ctx     context.Context
	Cmd     string
	Warning string
	Reason  string
	Client  llm.Client
	Lang    string
	Out     io.Writer
	In      io.Reader
}

// Ask renders the §2.4 confirmation UI and loops until the user picks
// execute (y) or cancel (n). Edit (e) re-runs stage-1 check only;
// explain (?) calls Client.Explain and re-renders.
func Ask(p Prompt) (Action, string, error) {
	if p.Ctx == nil {
		p.Ctx = context.Background()
	}
	scanner := bufio.NewScanner(p.In)
	cmd := p.Cmd
	warning := p.Warning
	reason := p.Reason

	for {
		render(p.Out, cmd, warning, reason)

		if !scanner.Scan() {
			return Cancel, "", scanner.Err()
		}
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "y", "Y":
			return Execute, cmd, nil
		case "n", "N":
			return Cancel, "", nil
		case "e", "E":
			fmt.Fprint(p.Out, "수정할 명령 입력: ")
			if !scanner.Scan() {
				return Cancel, "", scanner.Err()
			}
			newCmd := strings.TrimSpace(scanner.Text())
			if newCmd == "" {
				continue
			}
			v, r := safety.Check(newCmd)
			cmd = newCmd
			if v == safety.Blocked {
				fmt.Fprintf(p.Out, "\n차단됨: %s\n", r)
				return Cancel, "", nil
			} else if v >= safety.Warn {
				warning = "이 명령은 파괴적이며 되돌릴 수 없습니다."
				reason = r
			} else {
				warning = ""
				reason = ""
			}
		case "?":
			if p.Client == nil {
				fmt.Fprintln(p.Out, "\n설명 기능을 사용하려면 LLM 클라이언트가 필요합니다.")
				continue
			}
			explanation, err := p.Client.Explain(p.Ctx, cmd, p.Lang)
			if err != nil {
				fmt.Fprintf(p.Out, "\n설명 오류: %v\n", err)
			} else {
				fmt.Fprintln(p.Out, "\n"+explanation)
			}
		}
	}
}

// render writes the SPEC §2.4 layout to w.
func render(w io.Writer, cmd, warning, reason string) {
	red, cyan, dim, reset := termColors(w)
	fmt.Fprintf(w, "\n번역된 명령어: %s%s%s\n", cyan, cmd, reset)
	if warning != "" {
		fmt.Fprintf(w, "\n%s⚠  %s%s\n", red, warning, reset)
		if reason != "" {
			fmt.Fprintf(w, "   이유: %s\n", reason)
		}
	}
	fmt.Fprintf(w, "\n선택:\n")
	fmt.Fprintf(w, "%s  [y] 그래도 실행\n  [n] 취소\n  [e] 명령 직접 수정\n  [?] 설명 보기%s\n", dim, reset)
	fmt.Fprint(w, "\n선택 [y/n/e/?]: ")
}

func termColors(w io.Writer) (red, cyan, dim, reset string) {
	if os.Getenv("NO_COLOR") != "" {
		return "", "", "", ""
	}
	if f, ok := w.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return "\033[31m", "\033[36m", "\033[2m", "\033[0m"
	}
	return "", "", "", ""
}
