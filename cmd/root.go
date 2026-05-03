package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/biddan606/asksh/internal/config"
	shellctx "github.com/biddan606/asksh/internal/context"
	"github.com/biddan606/asksh/internal/executor"
	"github.com/biddan606/asksh/internal/history"
	"github.com/biddan606/asksh/internal/llm"
	"github.com/biddan606/asksh/internal/prompt"
	"github.com/biddan606/asksh/internal/safety"
	"github.com/spf13/cobra"
)

const version = "0.1.0-dev"

func Execute() {
	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load:", err)
		os.Exit(4)
	}
	factory := func(backend string) (llm.Client, error) {
		return newClientForBackend(cfg, backend)
	}
	if err := NewRootCmd(cfg, factory).Execute(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newClientForBackend(cfg config.Config, backend string) (llm.Client, error) {
	switch backend {
	case "openai":
		if cfg.OpenAI.APIKey == "" {
			return nil, fmt.Errorf("openai: api_key not set in config")
		}
		return llm.NewOpenAIClient(cfg.OpenAI.APIKey, cfg.OpenAI.Model, ""), nil
	default:
		return llm.NewOllamaClient(cfg.Ollama.Host, cfg.Ollama.Model), nil
	}
}

func NewRootCmd(cfg config.Config, newClient func(string) (llm.Client, error)) *cobra.Command {
	var dryRun bool
	var useOllama bool
	var useOpenAI bool

	root := &cobra.Command{
		Use:   "asksh <query>",
		Short: "Translate natural language into shell commands",
		Long: `asksh translates natural language queries (Korean or English) into shell
commands and runs them after interactive safety confirmation.`,
		Example: `  # 한국어 쿼리
  asksh "현재 디렉토리의 .log 파일 모두 삭제"

  # English query
  asksh "delete all .log files in current directory"

  # Translate only, no execution
  asksh --dry-run "list all git branches"

  # Configure backend
  asksh config`,
		Version:      version,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			backend := cfg.Backend.Default
			if useOllama {
				backend = "ollama"
			} else if useOpenAI {
				backend = "openai"
			}

			sc, err := shellctx.Collect()
			if err != nil {
				return err
			}
			query := strings.Join(args, " ")

			if dryRun {
				fmt.Fprintln(out, "query:", query)
				fmt.Fprintf(out, "cwd=%s os=%s shell=%s backend=%s\n", sc.CWD, sc.OS, sc.Shell, backend)

				ruleVerdict, ruleReason := safety.Check(query)
				if ruleReason != "" {
					fmt.Fprintf(out, "rule:  %s (%s)\n", strings.ToUpper(ruleVerdict.String()), ruleReason)
				} else {
					fmt.Fprintf(out, "rule:  %s\n", strings.ToUpper(ruleVerdict.String()))
				}

				if ruleVerdict == safety.Blocked {
					return fmt.Errorf("BLOCKED: %s", ruleReason)
				}

				if newClient != nil {
					if client, err := newClient(backend); err == nil {
						r := <-safety.Probe(cmd.Context(), client, query)
						if r.Err != nil {
							fmt.Fprintf(out, "llm:   unavailable (%s)\n", r.Err)
						} else if r.Reason != "" {
							fmt.Fprintf(out, "llm:   %s (%s)\n", r.Verdict.String(), r.Reason)
						} else {
							fmt.Fprintf(out, "llm:   %s\n", r.Verdict.String())
						}
					}
				}

				return nil
			}

			if newClient == nil {
				return fmt.Errorf("no LLM client available")
			}
			client, err := newClient(backend)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			translated, err := client.Translate(cmd.Context(), query, sc)
			if err != nil {
				return fmt.Errorf("translate: %w", err)
			}

			// Stage 1: rule-based safety check on translated command
			ruleVerdict, ruleReason := safety.Check(translated)
			if ruleVerdict == safety.Blocked {
				return fmt.Errorf("BLOCKED: %s", ruleReason)
			}

			// Stage 2: async LLM safety probe
			combined, combinedReason := ruleVerdict, ruleReason
			if cfg.Safety.ExtraLLMCheck {
				r := <-safety.Probe(cmd.Context(), client, translated)
				if r.Err != nil {
					fmt.Fprintf(out, "경고: LLM 안전성 검사 실패 (%v), 규칙 기반 결과만 사용\n", r.Err)
				} else if r.Verdict == safety.Dangerous {
					return fmt.Errorf("BLOCKED (LLM): %s", r.Reason)
				} else if r.Verdict > combined {
					combined = r.Verdict
					combinedReason = r.Reason
				}
			}

			// Show confirmation prompt if command is risky or always-confirm is set
			if combined >= safety.Warn || cfg.Safety.RequireConfirmation {
				var warning string
				if combined >= safety.Warn {
					warning = "이 명령은 파괴적이며 되돌릴 수 없습니다."
				}
				p := prompt.Prompt{
					Ctx:     cmd.Context(),
					Cmd:     translated,
					Warning: warning,
					Reason:  combinedReason,
					Client:  client,
					Lang:    llm.DetectLang(query),
					Out:     out,
					In:      cmd.InOrStdin(),
				}
				action, finalCmd, promptErr := prompt.Ask(p)
				if promptErr != nil {
					return promptErr
				}
				if action == prompt.Cancel {
					if cfg.History.Enable {
						_ = history.Append(history.Path(), history.Entry{
							Time:    time.Now().UTC().Format(time.RFC3339),
							Query:   query,
							Command: translated,
							Verdict: combined.String(),
							Result:  "cancelled",
						})
					}
					return nil
				}
				translated = finalCmd
			} else {
				fmt.Fprintf(out, "번역된 명령어: %s\n", translated)
			}

			runErr := executor.Run(cmd.Context(), sc.Shell, translated, out, cmd.ErrOrStderr(), cmd.InOrStdin())
			if cfg.History.Enable {
				result := "ok"
				if runErr != nil {
					result = "error: " + runErr.Error()
				}
				_ = history.Append(history.Path(), history.Entry{
					Time:    time.Now().UTC().Format(time.RFC3339),
					Query:   query,
					Command: translated,
					Verdict: combined.String(),
					Result:  result,
				})
			}
			return runErr
		},
	}

	root.Flags().BoolVar(&dryRun, "dry-run", false, "Print query and context info, skip LLM call")
	root.Flags().BoolVar(&useOllama, "ollama", false, "Force Ollama backend")
	root.Flags().BoolVar(&useOpenAI, "openai", false, "Force OpenAI backend")
	root.MarkFlagsMutuallyExclusive("ollama", "openai")

	root.AddCommand(newConfigCmd())

	return root
}

