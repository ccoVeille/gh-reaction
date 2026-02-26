package cmd

import (
	"context"
	"fmt"

	"github.com/ccoVeille/gh-reaction/internal/gh"
	"github.com/spf13/cobra"
)

// NewRepositoryCmd creates and returns the repository subcommand
func NewRepositoryCmd() *cobra.Command {
	repoCmd := &cobra.Command{
		Use:          "repository [owner/repo]",
		Short:        "Analyze reactions on repository items",
		Long:         "Analyze GitHub reactions on issues, PRs, and comments in a repository.",
		Args:         cobra.MaximumNArgs(1),
		RunE:         runRepositoryCmd,
		SilenceUsage: true,
	}

	return repoCmd
}

func runRepositoryCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	since, err := ParseSinceFlag()
	if err != nil {
		return err
	}

	var repo *gh.Repository

	// Check for repository flag or argument
	if len(args) > 0 {
		repo, err = gh.ParseRepositoryPath(args[0])
		if err != nil {
			return err
		}
	} else {
		repo, err = gh.CurrentRepository()
		if err != nil {
			return fmt.Errorf("use either a repository argument or run this command in a folder with a git repository: %w", err)
		}
	}

	reactions, err := gh.QueryRepository(ctx, since, *repo)
	if err != nil {
		return err
	}

	renderReactions(since, reactions, "reactions on repository items")
	return nil
}
