package cmd

import (
	"github.com/ccoVeille/gh-reaction/internal/app"
	"github.com/spf13/cobra"
)

// NewUserCmd creates and returns the user subcommand
func NewUserCmd() *cobra.Command {
	userCmd := &cobra.Command{
		Use:   "user [username]",
		Short: "Analyze reactions on posts made by a user",
		Long:  "Analyze GitHub reactions on posts made by a user across all repositories.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunUser(cmd.Context(), sinceFlag, args)
		},
		SilenceUsage: true,
	}

	addSinceFlagToCmd(userCmd)

	return userCmd
}
