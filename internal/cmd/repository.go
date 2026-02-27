package cmd

import (
	"github.com/ccoVeille/gh-reaction/internal/app"
	"github.com/spf13/cobra"
)

// NewRepositoryCmd creates and returns the repository subcommand
func NewRepositoryCmd() *cobra.Command {
	repoCmd := &cobra.Command{
		Use:   "repository [owner/repo]",
		Short: "Analyze reactions on repository items",
		Long:  "Analyze GitHub reactions on issues, PRs, and comments in a repository.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunRepository(cmd.Context(), sinceFlag, args)
		},
		SilenceUsage: true,
	}

	addSinceFlagToCmd(repoCmd)

	return repoCmd
}
