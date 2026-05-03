package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const version = "0.1.0-dev"

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	var dryRun bool
	var useOllama bool
	var useOpenAI bool

	root := &cobra.Command{
		Use:     "asksh <query>",
		Short:   "Translate natural language into shell commands",
		Version: version,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			if dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "query:", query)
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "[execution not yet implemented]")
			return nil
		},
	}

	root.Flags().BoolVar(&dryRun, "dry-run", false, "Translate only, do not execute")
	root.Flags().BoolVar(&useOllama, "ollama", false, "Force Ollama backend")
	root.Flags().BoolVar(&useOpenAI, "openai", false, "Force OpenAI backend")

	root.AddCommand(newConfigCmd())

	return root
}
