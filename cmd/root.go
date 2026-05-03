package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/biddan606/asksh/internal/config"
	shellctx "github.com/biddan606/asksh/internal/context"
	"github.com/biddan606/asksh/internal/llm"
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
		Use:     "asksh <query>",
		Short:   "Translate natural language into shell commands",
		Version: version,
		Args:    cobra.MinimumNArgs(1),
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
			fmt.Fprintln(out, translated)
			return nil
		},
	}

	root.Flags().BoolVar(&dryRun, "dry-run", false, "Print query and context info, skip LLM call")
	root.Flags().BoolVar(&useOllama, "ollama", false, "Force Ollama backend")
	root.Flags().BoolVar(&useOpenAI, "openai", false, "Force OpenAI backend")
	root.MarkFlagsMutuallyExclusive("ollama", "openai")

	root.AddCommand(newConfigCmd())

	return root
}
