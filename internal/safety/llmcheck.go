package safety

import (
	"context"

	"github.com/biddan606/asksh/internal/llm"
)

// ProbeResult holds the outcome of an async LLM safety check.
type ProbeResult struct {
	Verdict Verdict
	Reason  string
	Err     error
}

// Probe runs c.SafetyCheck in a goroutine and returns a channel that receives
// exactly one ProbeResult. The channel is buffered so the goroutine never blocks.
func Probe(ctx context.Context, c llm.Client, cmd string) <-chan ProbeResult {
	ch := make(chan ProbeResult, 1)
	go func() {
		v, reason, err := c.SafetyCheck(ctx, cmd)
		if err != nil {
			ch <- ProbeResult{Err: err}
			return
		}
		ch <- ProbeResult{Verdict: llmToSafety(v), Reason: reason}
	}()
	return ch
}

func llmToSafety(v llm.Verdict) Verdict {
	switch v {
	case llm.VerdictSafe:
		return Safe
	case llm.VerdictWarn:
		return Warn
	case llm.VerdictDangerous:
		return Dangerous
	default:
		return Warn
	}
}
