package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// RunWizard runs an interactive configuration wizard, reading answers from r,
// writing prompts to w, and saving the result to path.
func RunWizard(r io.Reader, w io.Writer, path string) error {
	cfg, err := Load(path)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	scanner := bufio.NewScanner(r)
	var scanErr error

	prompt := func(label, current string) string {
		fmt.Fprintf(w, "%s [%s]: ", label, current)
		if !scanner.Scan() {
			if e := scanner.Err(); e != nil && scanErr == nil {
				scanErr = e
			}
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
			if e := scanner.Err(); e != nil && scanErr == nil {
				scanErr = e
			}
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

	promptChoice := func(label, current string, choices []string) string {
		for {
			fmt.Fprintf(w, "%s [%s]: ", label, current)
			if !scanner.Scan() {
				if e := scanner.Err(); e != nil && scanErr == nil {
					scanErr = e
				}
				return current
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				return current
			}
			for _, c := range choices {
				if line == c {
					return line
				}
			}
			fmt.Fprintf(w, "  must be one of: %s\n", strings.Join(choices, "|"))
		}
	}

	// promptKey reads a secret without echoing when r is a real tty; falls back
	// to plain scanner (for tests and piped input).
	promptKey := func(label, current string) string {
		fmt.Fprintf(w, "%s (leave blank to keep existing): ", label)
		if f, ok := r.(*os.File); ok {
			key, err := term.ReadPassword(int(f.Fd()))
			fmt.Fprintln(w)
			if err == nil && len(key) > 0 {
				return string(key)
			}
			return current
		}
		if !scanner.Scan() {
			if e := scanner.Err(); e != nil && scanErr == nil {
				scanErr = e
			}
			return current
		}
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			return line
		}
		return current
	}

	cfg.Backend.Default = promptChoice("backend.default", cfg.Backend.Default, []string{"ollama", "openai"})
	cfg.Ollama.Host = prompt("ollama.host", cfg.Ollama.Host)
	cfg.Ollama.Model = prompt("ollama.model", cfg.Ollama.Model)
	cfg.OpenAI.APIKey = promptKey("openai.api_key", cfg.OpenAI.APIKey)
	cfg.OpenAI.Model = prompt("openai.model", cfg.OpenAI.Model)
	cfg.Safety.RequireConfirmation = promptBool("safety.require_confirmation", cfg.Safety.RequireConfirmation)
	cfg.Safety.ExtraLLMCheck = promptBool("safety.extra_llm_check", cfg.Safety.ExtraLLMCheck)
	cfg.History.Enable = promptBool("history.enable", cfg.History.Enable)

	if scanErr != nil {
		return fmt.Errorf("read input: %w", scanErr)
	}

	if err := Save(path, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(w, "\nConfiguration saved to %s\n", path)
	return nil
}
