package app

import (
	"context"
	"fmt"
	"time"

	"github.com/ccoveille/gh-reaction/internal/gh"
	"github.com/ccoveille/gh-reaction/internal/timeago"
)

const DefaultSinceDaysAgo = 90

func parseSinceFlag(sinceFlag string) (timeago.RelativeDate, error) {
	var since timeago.RelativeDate

	if sinceFlag == "" {
		since = timeago.NewRelativeDate(time.Now().AddDate(0, 0, -DefaultSinceDaysAgo))
	} else {
		if err := since.Set(sinceFlag); err != nil {
			return since, err
		}
	}

	since.Time = since.Time.Truncate(time.Hour).UTC()
	return since, nil
}

func displayAPIStats(ctx context.Context, result gh.QueryResult) {
	fmt.Printf("\nGraphQL cost: %d", result.Cost)
	rateLimit, err := gh.FetchRateLimit(ctx)
	if err == nil {
		fmt.Printf(" | Remaining: %d/%d", rateLimit.Remaining, rateLimit.Limit)
	}
	fmt.Println()
}

func RunUser(ctx context.Context, sinceFlag string, args []string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	since, err := parseSinceFlag(sinceFlag)
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

		if err := client.Get(ctx, "user", &me); err != nil {
			return err
		}
		author = me.Login
	}

	result, err := gh.QueryUser(ctx, since, author)
	if err != nil {
		return err
	}

	displayAPIStats(ctx, result)
	fmt.Println()
	renderReactions(since, result, "messages with reactions")
	return nil
}

func RunRepository(ctx context.Context, sinceFlag string, args []string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	since, err := parseSinceFlag(sinceFlag)
	if err != nil {
		return err
	}

	var repo *gh.Repository

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

	result, err := gh.QueryRepository(ctx, since, *repo)
	if err != nil {
		return err
	}

	displayAPIStats(ctx, result)
	fmt.Println()
	renderReactions(since, result, "reactions on repository items")
	return nil
}
