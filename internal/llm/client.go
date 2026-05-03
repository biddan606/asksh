package llm

import (
	"context"

	shellctx "github.com/biddan606/asksh/internal/context"
)

type Verdict string

const (
	VerdictSafe      Verdict = "safe"
	VerdictWarn      Verdict = "warn"
	VerdictDangerous Verdict = "dangerous"
)

type Client interface {
	Translate(ctx context.Context, query string, sc shellctx.ShellContext) (string, error)
	SafetyCheck(ctx context.Context, cmd string) (Verdict, string, error)
	Explain(ctx context.Context, cmd, lang string) (string, error)
}
