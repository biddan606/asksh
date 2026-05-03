package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/biddan606/asksh/internal/config"
	shellctx "github.com/biddan606/asksh/internal/context"
	"github.com/spf13/cobra"
)

const version = "0.1.0-dev"

func Execute() {
	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "config load:", err)
		os.Exit(4)
	}
	if err := NewRootCmd(cfg).Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCmd(cfg config.Config) *cobra.Command {
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
			if !dryRun {
				fmt.Fprintln(out, "[execution not yet implemented]")
				return nil
			}

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
			fmt.Fprintln(out, "query:", query)
			fmt.Fprintf(out, "cwd=%s os=%s shell=%s backend=%s\n", sc.CWD, sc.OS, sc.Shell, backend)
			return nil
		},
	}

	root.Flags().BoolVar(&dryRun, "dry-run", false, "Translate only, do not execute")
	root.Flags().BoolVar(&useOllama, "ollama", false, "Force Ollama backend")
	root.Flags().BoolVar(&useOpenAI, "openai", false, "Force OpenAI backend")
	root.MarkFlagsMutuallyExclusive("ollama", "openai")

	root.AddCommand(newConfigCmd())

	return root
}
