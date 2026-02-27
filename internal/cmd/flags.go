package cmd

import (
	"fmt"

	"github.com/ccoVeille/gh-reaction/internal/app"
	"github.com/spf13/cobra"
)

// addSinceFlagToCmd adds the since flag to a command to avoid duplication
func addSinceFlagToCmd(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&sinceFlag,
		"since",
		"",
		fmt.Sprintf(`Fetch messages since this date (e.g., "2023-01-02", "2h", "15m", "3d" ...) (default "%dd")`, app.DefaultSinceDaysAgo),
	)
}
