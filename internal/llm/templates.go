package llm

import (
	_ "embed"
	"text/template"
)

//go:embed prompts/translate.tmpl
var translateTmplSrc string

//go:embed prompts/safety.tmpl
var safetyTmplSrc string

var (
	translateTmpl = template.Must(template.New("translate").Parse(translateTmplSrc))
	safetyTmpl    = template.Must(template.New("safety").Parse(safetyTmplSrc))
)

// TranslateData holds the variables injected into translate.tmpl.
type TranslateData struct {
	CWD, OS, Shell, Lang, Query string
}

// SafetyData holds the variable injected into safety.tmpl.
type SafetyData struct {
	Cmd string
}

func TranslateTemplate() *template.Template { return translateTmpl }
func SafetyTemplate() *template.Template    { return safetyTmpl }
