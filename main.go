package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"

	"github.com/ccoVeille/gh-reaction/internal/gh"
	"github.com/ccoVeille/gh-reaction/internal/github"
	"github.com/ccoVeille/gh-reaction/internal/spinner"
)

type PostType string

const (
	PostTypeIssue       PostType = "issue"
	PostTypePullRequest PostType = "pull_request"
	PostTypeComment     PostType = "comment"
)

type Post struct {
	Type    PostType
	Date    github.Time
	Content string
	Author  github.User
	Link    string
	ID      string
}

func (a Post) FetchReactions(ctx context.Context, client *gh.RESTClient, repo gh.Repository) (Reactions, error) {
	var url string
	if a.Type == PostTypeComment {
		url = fmt.Sprintf("repos/%s/%s/issues/comments/%s/reactions?per_page=100", repo.Owner, repo.Name, a.ID)
	} else {
		url = fmt.Sprintf("repos/%s/%s/issues/%s/reactions?per_page=100", repo.Owner, repo.Name, a.ID)
	}

	var reactions []github.Reaction
	if err := client.Get(ctx, url, &reactions); err != nil {
		return nil, err
	}

	var results Reactions
	for _, reaction := range reactions {
		results = append(results, ReactionTo{
			Post:     a,
			Reaction: reaction,
		})
	}

	return results, nil
}

func (a Post) ContentPreview() string {
	content := a.Content
	for _, l := range strings.Split(content, "\n") {
		if strings.HasPrefix(l, ">") {
			// Skip quoted lines
			continue
		}

		l = strings.TrimSpace(l)
		if l == "" {
			// Skip empty lines
			continue
		}

		content = l
		break
	}

	const maxLen = 100
	if content == a.Content {
		return truncateString(content, maxLen)
	}
	return truncateString(content, maxLen)
}

func cleanString(content string) string {
	return strings.Trim(content, ".,… \n\r\t")
}

func truncateString(content string, maxLen int) string {
	content = cleanString(content)
	if len(content) < maxLen {
		return content
	}

	lastSpaceIdx := strings.LastIndex(content[:maxLen], " ")
	if lastSpaceIdx != -1 {
		content = cleanString(content[:lastSpaceIdx])
		return content + " …"
	}

	str := []rune(content)
	return cleanString(string(str[:maxLen])) + " …"
}

// Fetch messages posted by the user in the current repository
func fetchPosts(ctx context.Context, client *gh.RESTClient, gitHubRepo gh.Repository, minDate github.Time) ([]Post, error) {
	var posts []Post

	// Fetch issues and PRs created by the user in the repository
	page := 1
	for {
		// TODO use github.Issue
		userIssues := []struct {
			Title       string      `json:"title"`
			UpdatedAt   github.Time `json:"updated_at"`
			Author      github.User `json:"user"`
			PullRequest *struct{}   `json:"pull_request,omitempty"`
			Number      int         `json:"number"`
		}{}

		q := url.Values{
			"page":      []string{fmt.Sprint(page)},
			"per_page":  []string{"100"},
			"sort":      []string{"commented"},
			"direction": []string{"desc"},
		}

		if !minDate.IsZero() {
			q.Set("since", minDate.Format(time.RFC3339))
		}

		url := fmt.Sprintf("repos/%s/%s/issues?%s", gitHubRepo.Owner, gitHubRepo.Name, q.Encode())
		if err := client.Get(ctx, url, &userIssues); err != nil {
			return nil, err
		}
		if len(userIssues) == 0 {
			break
		}
		for _, issue := range userIssues {
			postType := PostTypeIssue
			if issue.PullRequest != nil {
				postType = PostTypePullRequest
			}

			posts = append(posts, Post{
				Type:    postType,
				Date:    issue.UpdatedAt,
				Content: issue.Title,
				Author:  issue.Author,
				Link:    fmt.Sprintf("https://github.com/%s/%s/issues/%d", gitHubRepo.Owner, gitHubRepo.Name, issue.Number),
				ID:      fmt.Sprintf("%d", issue.Number),
			})
		}
		page++
	}

	// Fetch comments made by the user in the repository
	page = 1
	for {

		// TODO use github.Comment
		userComments := []struct {
			Body      string      `json:"body"`
			UpdatedAt github.Time `json:"updated_at"`
			Author    github.User `json:"user"`
			Link      string      `json:"html_url"`
			ID        int         `json:"id"`
		}{}

		q := url.Values{
			"per_page":  []string{"100"},
			"sort":      []string{"updated"},
			"direction": []string{"desc"},
		}
		if !minDate.IsZero() {
			q.Set("since", minDate.Format(time.RFC3339))
		}

		url := fmt.Sprintf("repos/%s/%s/issues/comments?page=%d&%s", gitHubRepo.Owner, gitHubRepo.Name, page, q.Encode())
		if err := client.Get(ctx, url, &userComments); err != nil {
			return nil, err
		}
		if len(userComments) == 0 {
			break
		}
		for _, comment := range userComments {
			posts = append(posts, Post{
				Type:    PostTypeComment,
				Date:    comment.UpdatedAt,
				Content: comment.Body,
				Author:  comment.Author,
				Link:    comment.Link,
				ID:      fmt.Sprintf("%d", comment.ID),
			})
		}
		page++
	}

	// Sort posts by time in descending order
	slices.SortFunc(posts, func(a1, a2 Post) int {
		return a2.Date.Compare(a1.Date.Time)
	})

	return posts, nil
}

type Reactions []ReactionTo

func (r *Reactions) Append(reactions ...ReactionTo) {
	*r = append(*r, reactions...)
}

func (r *Reactions) Clean() {
	clean := slices.DeleteFunc(*r, func(r1 ReactionTo) bool {
		// filter out bot reactions
		return r1.Reaction.User.IsBot()
	})

	slices.SortFunc(clean, func(r1, r2 ReactionTo) int {
		return r1.Reaction.Created.Compare(r2.Reaction.Created.Time)
	})

	*r = clean
}

type ValueCount[T any] struct {
	Value T
	Count int
}

type ValueCounts[T any] []ValueCount[T]

func (v ValueCounts[T]) Top(nb int) ValueCounts[T] {
	if nb <= 0 {
		return nil
	}

	// Sort the values by count (descending)
	slices.SortFunc(v, func(a, b ValueCount[T]) int {
		if a.Count == b.Count {
			return cmp.Compare(fmt.Sprint(a.Value), fmt.Sprint(b.Value))
		}

		return b.Count - a.Count
	})

	// Return the top N values
	if nb > len(v) {
		nb = len(v)
	}
	return v[:nb]
}

func (v ValueCounts[T]) MaxSizeCount() int {
	var m int
	for _, vc := range v {
		m = max(m, vc.Count)
	}

	return int(math.Ceil(math.Log10(float64(m))))
}

func (v ValueCounts[T]) MaxSizeValue(f func(T) string) int {
	var m int
	for _, vc := range v {
		s := f(vc.Value)
		m = max(m, len(s))
	}
	return m
}

func (r Reactions) Users() ValueCounts[github.User] {
	userCounts := make(map[string]ValueCount[github.User])

	for _, reaction := range r {
		if reaction.Reaction.User.Login == nil {
			continue
		}
		key := *reaction.Reaction.User.Login
		u, found := userCounts[key]
		if !found {
			u = ValueCount[github.User]{Value: reaction.Reaction.User}
		}
		u.Count++
		userCounts[key] = u
	}

	return slices.Collect(maps.Values(userCounts))
}

func (r Reactions) Posts() ValueCounts[Post] {
	postCounts := make(map[string]ValueCount[Post])

	for _, reaction := range r {
		key := reaction.Post.Link
		u, found := postCounts[key]
		if !found {
			u = ValueCount[Post]{Value: reaction.Post}
		}
		u.Count++
		postCounts[key] = u
	}

	return slices.Collect(maps.Values(postCounts))
}

func (r Reactions) Reactions() ValueCounts[string] {
	reactionCounts := make(map[string]ValueCount[string])

	for _, reaction := range r {
		key := reaction.Reaction.Type()
		u, found := reactionCounts[key]
		if !found {
			u = ValueCount[string]{Value: key}
		}
		u.Count++
		reactionCounts[key] = u
	}

	return slices.Collect(maps.Values(reactionCounts))
}

type ReactionTo struct {
	Reaction github.Reaction
	Post     Post
}

func run(ctx context.Context) error {
	client, err := gh.DefaultRESTClient()
	if err != nil {
		return err
	}

	// Fetch current user info
	var currentUser github.User
	if err := client.Get(ctx, "user", &currentUser); err != nil {
		return err
	}

	repo, err := gh.CurrentRepository()
	if err != nil {
		return err
	}

	logger := slog.Default()

	logger.Info("Looking for reactions on user posts",
		"repository_owner", repo.Owner,
		"repository_name", repo.Name,
		"user", currentUser)

	var minDate github.Time
	minDate = github.Time{Time: time.Now().AddDate(0, -1, 0)}

	posts, err := fetchPosts(ctx, client, repo, minDate)
	if err != nil {
		return err
	}

	if len(posts) == 0 {
		logger.Warn("Fetched no recent posts", "since", minDate.String())
		return nil
	}

	logger.Info("Fetched posts", "total", len(posts), "since", minDate.String())

	// Keep only posts authored by the current user
	userPosts := slices.DeleteFunc(posts, func(a Post) bool {
		if a.Author.Login == nil || currentUser.Login == nil {
			return false
		}

		return *a.Author.Login != *currentUser.Login
	})

	var allReactions Reactions

	logger.Info("Fetched user posts", "total", len(userPosts), "since", minDate.String(), "user", currentUser)

	const maxUserPost = 100
	if len(userPosts) > maxUserPost {
		userPosts = userPosts[:maxUserPost]
		logger.Warn("Truncated user posts", "max_size", maxUserPost)
		logger.Info("Fetched user posts", "total", len(userPosts), "since", minDate.String(), "user", currentUser)
	}

	spinner := spinner.New("fetched %d post reactions…")
	spinner.Start(ctx, os.Stdout)
	for _, post := range userPosts {
		spinner.Inc()
		reactions, err := post.FetchReactions(ctx, client, repo)
		if err != nil {
			return err
		}
		allReactions.Append(reactions...)
		logger.Debug("Fetched reactions for post", "total", len(reactions), "link", post.Link)
	}
	spinner.Done()

	allReactions.Clean()

	if len(allReactions) == 0 {
		logger.Info("No reactions found on the period", "since", minDate.String(), "user", currentUser)
		return nil
	}

	fmt.Println("Since:", minDate)
	fmt.Println("Total messages:", len(posts))
	fmt.Println("Total messages from user:", len(userPosts))
	fmt.Println()

	postsWithReactions := allReactions.Posts()
	fmt.Println("Total messages with reactions:", len(postsWithReactions))
	topPosts := postsWithReactions.Top(5)
	if len(postsWithReactions) > len(topPosts) {
		fmt.Println("Top message with reactions:")
	}

	maxSizeCount := topPosts.MaxSizeCount()
	for _, post := range topPosts {
		fmt.Printf("%.*d reactions: %s\n", maxSizeCount, post.Count, post.Value.Link)
	}
	fmt.Println()

	fmt.Println("Total reactions:", len(allReactions))
	topReactions := allReactions.Reactions()
	fmt.Print("Top reactions: ")
	maxSizeCount = topReactions.MaxSizeCount()

	for i, reaction := range topReactions {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%.*d %s", maxSizeCount, reaction.Count, reaction.Value)
	}
	fmt.Print("\n\n")

	users := allReactions.Users()
	fmt.Println("Total users who reacted:", len(users))
	topUsers := users.Top(5)
	if len(users) > len(topUsers) {
		fmt.Println("Top users who reacted:")
	}

	maxSizeCount = topUsers.MaxSizeCount()
	maxSizeLogin := topUsers.MaxSizeValue(func(u github.User) string {
		if u.Login == nil {
			return ""
		}
		return *u.Login
	})

	for _, user := range topUsers {
		fmt.Printf("%.*d %-*s %s\n", maxSizeCount, user.Count, maxSizeLogin, user.Value, user.Value.GitHubURL())
	}
	fmt.Println()

	for _, reaction := range allReactions {

		fmt.Printf("%s %s reacted with %s to:\n",
			reaction.Reaction.Created,
			reaction.Reaction.User,
			reaction.Reaction.Type())

		fmt.Printf("  Message: %s\n", reaction.Post.ContentPreview())
		fmt.Printf("  Message Type: %s\n", reaction.Post.Type)
		fmt.Printf("  Author: %s\n", reaction.Post.Author)
		fmt.Printf("  Posted: %s\n", reaction.Post.Date)
		fmt.Printf("  Link: %s\n", reaction.Post.Link)
		fmt.Println()
	}

	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err := run(ctx)
	switch {
	case errors.Is(err, context.Canceled) && ctx.Err() != nil:
		// handle the CTRL+C case silently
		os.Exit(130) // classic exit code for a SIGINT (Ctrl+C) termination

	case err != nil:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1) // return a non-zero exit code for any other error
	}
}
