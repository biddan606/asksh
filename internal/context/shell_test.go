package shellctx_test

import (
	"fmt"
	"strings"
	"testing"

	shellctx "github.com/biddan606/asksh/internal/context"
)

func TestCollect_CWD(t *testing.T) {
	ctx, err := shellctx.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if ctx.CWD == "" {
		t.Error("CWD should not be empty")
	}
}

func TestCollect_Shell(t *testing.T) {
	ctx, err := shellctx.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if ctx.Shell == "" {
		t.Error("Shell should not be empty")
	}
	if strings.Contains(ctx.Shell, "/") {
		t.Errorf("Shell should be basename only, got: %q", ctx.Shell)
	}
}

func TestCollect_LangEmpty(t *testing.T) {
	ctx, err := shellctx.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if ctx.Lang != "" {
		t.Errorf("Lang should be empty until P3, got: %q", ctx.Lang)
	}
}

func TestCollect_OSNonEmpty(t *testing.T) {
	ctx, err := shellctx.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if ctx.OS == "" {
		t.Error("OS should not be empty")
	}
}

func TestCollectWith_InjectsOSVersion(t *testing.T) {
	want := "TestOS 99.0"
	ctx, err := shellctx.CollectWith(func() (string, error) {
		return want, nil
	})
	if err != nil {
		t.Fatalf("CollectWith returned error: %v", err)
	}
	if ctx.OS != want {
		t.Errorf("OS = %q, want %q", ctx.OS, want)
	}
}

func TestCollectWith_FallsBackToGOOS(t *testing.T) {
	ctx, err := shellctx.CollectWith(func() (string, error) {
		return "", fmt.Errorf("sw_vers failed")
	})
	if err != nil {
		t.Fatalf("CollectWith returned error: %v", err)
	}
	if ctx.OS == "" {
		t.Error("OS should fall back to runtime.GOOS, not be empty")
	}
}
