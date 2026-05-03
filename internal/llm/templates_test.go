package llm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/biddan606/asksh/internal/llm"
)

func TestTranslateTemplateEnglish(t *testing.T) {
	data := llm.TranslateData{
		CWD:   "/home/user/project",
		OS:    "darwin 14.0",
		Shell: "zsh",
		Lang:  "en",
		Query: "list all files",
	}
	var buf bytes.Buffer
	if err := llm.TranslateTemplate().Execute(&buf, data); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"/home/user/project", "darwin 14.0", "zsh", "list all files", "code fences"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestTranslateTemplateKorean(t *testing.T) {
	data := llm.TranslateData{
		CWD:   "/tmp",
		OS:    "darwin 14.0",
		Shell: "zsh",
		Lang:  "ko",
		Query: "파일 목록",
	}
	var buf bytes.Buffer
	if err := llm.TranslateTemplate().Execute(&buf, data); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"/tmp", "파일 목록", "코드 펜스"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestSafetyTemplateRendersCmd(t *testing.T) {
	data := llm.SafetyData{Cmd: "rm -rf /"}
	var buf bytes.Buffer
	if err := llm.SafetyTemplate().Execute(&buf, data); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"rm -rf /", "safe", "warn", "dangerous"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
