package llm_test

import (
	"testing"

	"github.com/biddan606/asksh/internal/llm"
)

func TestDetectLang(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  string
	}{
		{"pure korean", "파일 목록 보여줘", "ko"},
		{"pure english", "list files", "en"},
		{"mixed korean wins", "list .log 파일", "ko"},
		{"empty", "", "en"},
		{"numeric only", "12345", "en"},
		{"special chars", "!@#$%", "en"},
		{"japanese hiragana", "こんにちは", "en"},
		{"chinese", "文件列表", "en"},
		{"hangul jamo U+3130 block", "ㄱㄴㄷ", "en"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := llm.DetectLang(c.query)
			if got != c.want {
				t.Errorf("DetectLang(%q) = %q; want %q", c.query, got, c.want)
			}
		})
	}
}
