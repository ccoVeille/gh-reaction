package cmd

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ccoVeille/gh-reaction/internal/gh"
	"github.com/ccoVeille/gh-reaction/internal/github"
	"github.com/ccoVeille/gh-reaction/internal/timeago"
)

type postWithReactions struct {
	Post               Post
	Reactions          Reactions
	MostRecentReaction time.Time
}

func renderReactions(since timeago.RelativeDate, reactions []gh.ReactionResult, emptyLabel string) {
	fmt.Println()

	if len(reactions) == 0 {
		fmt.Println("Stats since", since)
		fmt.Println(len(reactions), emptyLabel)
		fmt.Println()
		return
	}

	allReactions := buildReactions(reactions)
	allReactions.Clean()

	fmt.Println()
	fmt.Println("Posts with reactions (sorted by most recent reaction, oldest -> newest):")

	postGroups := groupReactionsByPost(allReactions)
	for _, group := range postGroups {
		fmt.Print(group.Post.String())
		for _, reaction := range group.Reactions {
			fmt.Print(reaction.String())
		}
		fmt.Println()
	}

	topReactions := allReactions.Reactions().Top(10)

	var reacts []string
	for _, reaction := range topReactions {
		reacts = append(reacts, fmt.Sprintf("%s %d", reaction.Value, reaction.Count))
	}

	fmt.Printf("Total reactions: %d\n%s\n\n", len(allReactions), strings.Join(reacts, ", "))

	users := allReactions.Users()
	topUsers := users.Top(5)
	if len(users) > len(topUsers) {
		fmt.Printf("Top reactors (%d of %d):\n", len(topUsers), len(users))
	} else {
		fmt.Printf("Reactors (%d):\n", len(users))
	}

	maxSizeCount := topUsers.MaxSizeCount()
	maxSizeLogin := topUsers.MaxSizeValue(func(u github.User) string {
		return u.Login
	})

	for _, user := range topUsers {
		fmt.Printf("%*s %-*s %s\n", maxSizeCount, strconv.Itoa(user.Count), maxSizeLogin, user.Value, user.Value.GitHubURL())
	}
	fmt.Println()
}

func buildReactions(reactions []gh.ReactionResult) Reactions {
	var allReactions Reactions
	for _, reaction := range reactions {
		content := string(reaction.Parent.Title)
		if content == "" {
			content = string(reaction.Parent.Body)
		}

		allReactions = append(allReactions, ReactionTo{
			Reaction: github.Reaction{
				CreatedAt: github.Time{
					Time: reaction.CreatedAt,
				},
				Content: reaction.Content,
				User: github.User{
					Login:    reaction.User.Login,
					Name:     reaction.User.Name,
					Type:     reaction.User.Type,
					IsViewer: reaction.User.IsViewer,
				},
			},
			Post: Post{
				ID:   reaction.Parent.ID,
				Type: PostType(reaction.Type),
				Date: github.Time{Time: reaction.Parent.CreatedAt},
				Author: github.User{
					Login:    reaction.Parent.Author.Login,
					Name:     reaction.Parent.Author.Name,
					Type:     reaction.Parent.Author.Type,
					IsViewer: reaction.Parent.Author.IsViewer,
				},
				Content: content,
				Link:    reaction.Parent.URL,
			},
		})
	}

	return allReactions
}

func groupReactionsByPost(allReactions Reactions) []*postWithReactions {
	grouped := make(map[string]*postWithReactions)
	for _, reaction := range allReactions {
		key := reaction.Post.ID
		if _, exists := grouped[key]; !exists {
			grouped[key] = &postWithReactions{
				Post:               reaction.Post,
				MostRecentReaction: reaction.Reaction.CreatedAt.Time,
			}
		}
		grouped[key].Reactions = append(grouped[key].Reactions, reaction)
		if reaction.Reaction.CreatedAt.After(grouped[key].MostRecentReaction) {
			grouped[key].MostRecentReaction = reaction.Reaction.CreatedAt.Time
		}
	}

	var postGroups []*postWithReactions
	for _, group := range grouped {
		postGroups = append(postGroups, group)
	}
	slices.SortFunc(postGroups, func(a, b *postWithReactions) int {
		return a.MostRecentReaction.Compare(b.MostRecentReaction)
	})

	return postGroups
}
