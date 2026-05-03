package history_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biddan606/asksh/internal/history"
)

func TestAppend_CreatesFileWithRestrictedMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.log")

	err := history.Append(path, history.Entry{
		Time:    "2026-05-03T00:00:00Z",
		Query:   "list files",
		Command: "ls",
		Verdict: "safe",
		Result:  "ok",
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("file mode = %04o, want 0600", got)
	}
}

func TestAppend_WritesValidJSONLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.log")

	e := history.Entry{
		Time:    "2026-05-03T12:00:00Z",
		Query:   "list files",
		Command: "ls",
		Verdict: "safe",
		Result:  "ok",
	}
	if err := history.Append(path, e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got history.Entry
	if err := json.Unmarshal(data[:len(data)-1], &got); err != nil {
		t.Fatalf("unmarshal %q: %v", data, err)
	}
	if got != e {
		t.Errorf("got %+v, want %+v", got, e)
	}
}

func TestAppend_AccumulatesMultipleEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.log")

	entries := []history.Entry{
		{Time: "2026-05-03T10:00:00Z", Query: "q1", Command: "ls", Verdict: "safe", Result: "ok"},
		{Time: "2026-05-03T11:00:00Z", Query: "q2", Command: "pwd", Verdict: "safe", Result: "ok"},
	}
	for _, e := range entries {
		if err := history.Append(path, e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := splitLines(data)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	for i, line := range lines {
		var got history.Entry
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatalf("unmarshal line %d: %v", i, err)
		}
		if got != entries[i] {
			t.Errorf("line %d: got %+v, want %+v", i, got, entries[i])
		}
	}
}

func TestAppend_CreatesMissingParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "dir", "history.log")

	err := history.Append(path, history.Entry{
		Time: "2026-05-03T00:00:00Z", Query: "q", Command: "ls", Verdict: "safe", Result: "ok",
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestPath_DefaultUnderDotLocalShare(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	got := history.Path()
	if !strings.Contains(got, filepath.Join(".local", "share", "asksh", "history.log")) {
		t.Errorf("Path() = %q, want it to contain .local/share/asksh/history.log", got)
	}
}

func TestPath_XDGDataHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	want := filepath.Join(dir, "asksh", "history.log")
	if got := history.Path(); got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func splitLines(data []byte) []string {
	var lines []string
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, string(data[start:i]))
			}
			start = i + 1
		}
	}
	return lines
}
