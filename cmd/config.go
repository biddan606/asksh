package cmd

import (
	"os"

	"github.com/biddan606/asksh/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Interactive configuration wizard",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.RunWizard(os.Stdin, cmd.OutOrStdout(), config.ConfigPath())
		},
	}
}
