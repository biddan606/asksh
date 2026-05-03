package llm

import (
	"bytes"
	"context"
	"strings"

	shellctx "github.com/biddan606/asksh/internal/context"
)

func cleanCmd(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "```"); i != -1 {
		s = s[i+3:]
		if nl := strings.Index(s, "\n"); nl != -1 {
			s = s[nl+1:]
		}
		if end := strings.Index(s, "```"); end != -1 {
			s = s[:end]
		}
		s = strings.TrimSpace(s)
	}
	// Strip prefixes the LLM may add despite instructions.
	for _, pfx := range []string{"Command: ", "명령: ", "$ "} {
		if after, ok := strings.CutPrefix(s, pfx); ok {
			s = after
			break
		}
	}
	if strings.ContainsRune(s, '\n') {
		for _, line := range strings.Split(s, "\n") {
			if l := strings.TrimSpace(line); l != "" && !strings.HasPrefix(l, "#") {
				s = l
				break
			}
		}
	}
	return strings.TrimSpace(s)
}

func parseSafety(raw string) (Verdict, string, error) {
	raw = strings.TrimSpace(raw)
	word, reason, _ := strings.Cut(raw, " ")
	switch v := Verdict(strings.ToLower(word)); v {
	case VerdictSafe, VerdictWarn, VerdictDangerous:
		return v, reason, nil
	default:
		return VerdictWarn, raw, nil
	}
}

func translate(chat func(context.Context, string) (string, error), ctx context.Context, query string, sc shellctx.ShellContext) (string, error) {
	lang := DetectLang(query)
	var buf bytes.Buffer
	if err := TranslateTemplate().Execute(&buf, TranslateData{
		CWD:   sc.CWD,
		OS:    sc.OS,
		Shell: sc.Shell,
		Lang:  lang,
		Query: query,
	}); err != nil {
		return "", err
	}
	return chat(ctx, buf.String())
}

func checkSafety(chat func(context.Context, string) (string, error), ctx context.Context, cmd string) (Verdict, string, error) {
	var buf bytes.Buffer
	if err := SafetyTemplate().Execute(&buf, SafetyData{Cmd: cmd}); err != nil {
		return "", "", err
	}
	raw, err := chat(ctx, buf.String())
	if err != nil {
		return "", "", err
	}
	return parseSafety(raw)
}

func explain(chat func(context.Context, string) (string, error), ctx context.Context, cmd, lang string) (string, error) {
	if lang == "ko" {
		return chat(ctx, "다음 쉘 명령을 간단히 설명해 주세요:\n"+cmd)
	}
	return chat(ctx, "Briefly explain this shell command:\n"+cmd)
}
