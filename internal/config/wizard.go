package config

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// RunWizard runs an interactive configuration wizard, reading answers from r,
// writing prompts to w, and saving the result to path.
func RunWizard(r io.Reader, w io.Writer, path string) error {
	cfg, err := Load(path)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	scanner := bufio.NewScanner(r)

	prompt := func(label, current string) string {
		fmt.Fprintf(w, "%s [%s]: ", label, current)
		if !scanner.Scan() {
			return current
		}
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			return line
		}
		return current
	}

	promptBool := func(label string, current bool) bool {
		def := "y"
		if !current {
			def = "n"
		}
		fmt.Fprintf(w, "%s [%s]: ", label, def)
		if !scanner.Scan() {
			return current
		}
		switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
		case "y", "yes", "true":
			return true
		case "n", "no", "false":
			return false
		default:
			return current
		}
	}

	cfg.Backend.Default = prompt("backend.default (ollama|openai)", cfg.Backend.Default)
	cfg.Ollama.Host = prompt("ollama.host", cfg.Ollama.Host)
	cfg.Ollama.Model = prompt("ollama.model", cfg.Ollama.Model)
	cfg.OpenAI.APIKey = prompt("openai.api_key", cfg.OpenAI.APIKey)
	cfg.OpenAI.Model = prompt("openai.model", cfg.OpenAI.Model)
	cfg.Safety.RequireConfirmation = promptBool("safety.require_confirmation", cfg.Safety.RequireConfirmation)
	cfg.Safety.ExtraLLMCheck = promptBool("safety.extra_llm_check", cfg.Safety.ExtraLLMCheck)
	cfg.History.Enable = promptBool("history.enable", cfg.History.Enable)

	if err := Save(path, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(w, "\nConfiguration saved to %s\n", path)
	return nil
}
