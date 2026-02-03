package cmd

import (
	"fmt"
	"time"

	"github.com/ccoVeille/gh-reaction/internal/timeago"
	"github.com/spf13/cobra"
)

const defaultSinceDaysAgo = 90

// addSinceFlagToCmd adds the since flag to a command to avoid duplication
func addSinceFlagToCmd(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&sinceFlag,
		"since",
		"",
		fmt.Sprintf(`Fetch messages since this date (e.g., "2023-01-02", "2h", "15m", "3d" ...) (default "%dd")`, defaultSinceDaysAgo),
	)
}

// ParseSinceFlag parses the since flag and returns a RelativeDate
func ParseSinceFlag() (timeago.RelativeDate, error) {
	var since timeago.RelativeDate

	if sinceFlag == "" {
		since = timeago.NewRelativeDate(time.Now().AddDate(0, 0, -defaultSinceDaysAgo))
	} else {
		if err := since.Set(sinceFlag); err != nil {
			return since, err
		}
	}

	since.Time = since.Time.Truncate(time.Hour).UTC()
	return since, nil
}
