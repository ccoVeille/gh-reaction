package cmd

import (
	"github.com/spf13/cobra"
)

var (
	sinceFlag string
)

// NewRootCmd creates and returns the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "gh-reaction",
		Short:        "Analyze GitHub reactions on your posts",
		Long:         "A CLI tool to analyze GitHub reactions (emojis) on your or someone else's posts (issues, PRs, comments).",
		SilenceUsage: true,
	}

	// Add subcommands

	userCmd := NewUserCmd()
	repoCmd := NewRepositoryCmd()

	addSinceFlagToCmd(userCmd)
	addSinceFlagToCmd(repoCmd)

	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(repoCmd)

	return rootCmd
}
