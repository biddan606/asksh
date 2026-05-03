package safety_test

import (
	"context"
	"errors"
	"testing"

	shellctx "github.com/biddan606/asksh/internal/context"
	"github.com/biddan606/asksh/internal/llm"
	"github.com/biddan606/asksh/internal/safety"
)

type mockClient struct {
	verdict llm.Verdict
	reason  string
	err     error
	block   chan struct{} // if non-nil, SafetyCheck blocks until closed
}

func (m *mockClient) Translate(_ context.Context, _ string, _ shellctx.ShellContext) (string, error) {
	return "", nil
}

func (m *mockClient) SafetyCheck(ctx context.Context, _ string) (llm.Verdict, string, error) {
	if m.block != nil {
		select {
		case <-m.block:
		case <-ctx.Done():
			return "", "", ctx.Err()
		}
	}
	return m.verdict, m.reason, m.err
}

func (m *mockClient) Explain(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func TestProbeSafe(t *testing.T) {
	c := &mockClient{verdict: llm.VerdictSafe, reason: "looks fine"}
	ch := safety.Probe(context.Background(), c, "ls")
	r := <-ch
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.Verdict != safety.Safe {
		t.Errorf("verdict = %v; want Safe", r.Verdict)
	}
	if r.Reason != "looks fine" {
		t.Errorf("reason = %q; want %q", r.Reason, "looks fine")
	}
}

func TestProbeWarn(t *testing.T) {
	c := &mockClient{verdict: llm.VerdictWarn, reason: "risky op"}
	ch := safety.Probe(context.Background(), c, "chmod 777 /etc/passwd")
	r := <-ch
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.Verdict != safety.Warn {
		t.Errorf("verdict = %v; want Warn", r.Verdict)
	}
}

func TestProbeDangerous(t *testing.T) {
	c := &mockClient{verdict: llm.VerdictDangerous, reason: "deletes files"}
	ch := safety.Probe(context.Background(), c, "rm -rf ./tmp")
	r := <-ch
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.Verdict != safety.Dangerous {
		t.Errorf("verdict = %v; want Dangerous", r.Verdict)
	}
}

func TestProbeError(t *testing.T) {
	want := errors.New("llm unavailable")
	c := &mockClient{err: want}
	ch := safety.Probe(context.Background(), c, "ls")
	r := <-ch
	if r.Err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(r.Err, want) {
		t.Errorf("err = %v; want %v", r.Err, want)
	}
}

func TestProbeContextCancel(t *testing.T) {
	block := make(chan struct{})
	c := &mockClient{block: block}

	ctx, cancel := context.WithCancel(context.Background())
	ch := safety.Probe(ctx, c, "ls")
	cancel()

	r := <-ch
	if r.Err == nil {
		t.Fatal("expected error on context cancel")
	}
	if !errors.Is(r.Err, context.Canceled) {
		t.Errorf("err = %v; want context.Canceled", r.Err)
	}
}
