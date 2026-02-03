package cmd

import (
	"context"

	"github.com/ccoVeille/gh-reaction/internal/gh"
	"github.com/spf13/cobra"
)

// NewUserCmd creates and returns the user subcommand
func NewUserCmd() *cobra.Command {
	userCmd := &cobra.Command{
		Use:          "user [username]",
		Short:        "Analyze reactions on posts made by a user",
		Long:         "Analyze GitHub reactions on posts made by a user across all repositories.",
		Args:         cobra.MaximumNArgs(1),
		RunE:         runUserCmd,
		SilenceUsage: true,
	}

	return userCmd
}

func runUserCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	since, err := ParseSinceFlag()
	if err != nil {
		return err
	}

	var author string
	if len(args) > 0 {
		author = args[0]
	} else {
		client, err := gh.DefaultRESTClient()
		if err != nil {
			return err
		}

		var me struct {
			Login string `json:"login"`
		}

		// GET /user returns the authenticated user's profile
		if err := client.Get(ctx, "user", &me); err != nil {
			return err
		}
		author = me.Login
	}

	reactions, err := gh.QueryUser(ctx, since, author)
	if err != nil {
		return err
	}

	renderReactions(since, reactions, "messages with reactions")
	return nil
}
