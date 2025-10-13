// Package main implements a CLI tool to analyze GitHub reactions on your posts (issues, PRs, comments).
package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"maps"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ccoVeille/gh-reaction/internal/gh"
	"github.com/ccoVeille/gh-reaction/internal/github"
	"github.com/ccoVeille/gh-reaction/internal/spinner"
	"github.com/ccoVeille/gh-reaction/internal/timeago"
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
func fetchPosts(ctx context.Context, client *gh.RESTClient, gitHubRepo gh.Repository, minDate timeago.RelativeDate) ([]Post, error) {
	suffix := fmt.Sprintf("on github.com/%s/%s since %s", gitHubRepo.Owner, gitHubRepo.Name, minDate.String())

	fmt.Printf("Looking for posts %s\n", suffix)

	var posts []Post

	spin := spinner.New(os.Stdout)
	spin.Start(ctx, "fetching posts")

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
			spin.Progress("fetched %d posts", len(posts))
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
			spin.Progress("fetched %d posts", len(posts))
		}
		page++
	}

	spin.Done("✔️ fetched %d posts", len(posts))

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
		return r1.Reaction.CreatedAt.Compare(r2.Reaction.CreatedAt.Time)
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
		m = max(m, len(strconv.Itoa(vc.Count)))
	}

	return m
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

func (r Reactions) Authors() ValueCounts[github.User] {
	userCounts := make(map[string]ValueCount[github.User])

	for _, reaction := range r {
		if reaction.Post.Author.Login == nil {
			continue
		}
		key := *reaction.Post.Author.Login
		u, found := userCounts[key]
		if !found {
			u = ValueCount[github.User]{Value: reaction.Post.Author}
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

func (r ReactionTo) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Reacted on    %s (%s)\n", r.Reaction.CreatedAt.Format(time.DateOnly), r.Reaction.CreatedAt))
	sb.WriteString(fmt.Sprintf("Reacted by:   %s\n", r.Reaction.User))
	sb.WriteString(fmt.Sprintf("Reacted with: %s\n", r.Reaction.Type()))
	sb.WriteString(r.Post.String())
	return sb.String()
}

func (p Post) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Post message: %s\n", p.ContentPreview()))
	sb.WriteString(fmt.Sprintf("Post type:    %s\n", p.Type))
	sb.WriteString(fmt.Sprintf("Post author:  %s\n", p.Author))
	sb.WriteString(fmt.Sprintf("Post date:    %s\n", p.Date))
	sb.WriteString(fmt.Sprintf("Post link:    %s\n", p.Link))
	return sb.String()
}

func parseCLIOptions() (cliOptions, error) {
	var opts cliOptions
	fl := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	fl.StringVar(&opts.author, "author", "", "Limit to messages authored by this GitHub username")
	fl.IntVar(&opts.limit, "limit", 50, "Maximum number of messages to fetch")

	defaultSinceDaysAgo := 90
	fl.Var(&opts.since, "since", fmt.Sprintf(`Fetch messages since this date (e.g., "2023-01-02", "2h", "15m", "3d" ...) (default "%dd")`, defaultSinceDaysAgo))

	fl.Usage = func() {
		// add a simple --help flag
		fmt.Print("Available Flags:\n")
		fl.PrintDefaults()
	}
	err := fl.Parse(os.Args[1:])
	if err != nil {
		return opts, err
	}

	if opts.since.IsZero() {
		opts.since = timeago.NewRelativeDate(time.Now().AddDate(0, 0, -defaultSinceDaysAgo))
	}
	opts.since.Time = opts.since.Time.Truncate(time.Hour).UTC()

	return opts, nil
}

func run(ctx context.Context) error {
	client, err := gh.DefaultRESTClient()
	if err != nil {
		return err
	}

	repo, err := gh.CurrentRepository()
	if err != nil {
		return err
	}

	opts, err := parseCLIOptions()
	if err != nil {
		return err
	}

	since := opts.since

	allPosts, err := fetchPosts(ctx, client, repo, since)
	if err != nil {
		return err
	}

	if len(allPosts) == 0 {
		fmt.Println("\nNo posts found since ", since.String())
		return nil
	}

	posts := allPosts
	if opts.author != "" {
		// Keep only posts authored by the current user
		posts = slices.DeleteFunc(posts, func(a Post) bool {
			if a.Author.Login == nil {
				return false
			}

			return !strings.EqualFold(*a.Author.Login, opts.author)
		})
		fmt.Printf("Limited analysis to %d %s posts\n", len(posts), opts.author)
	}

	if opts.limit > 0 && len(posts) > opts.limit {
		posts = posts[:opts.limit]

		lastPost := posts[len(posts)-1]
		since = timeago.NewRelativeDate(lastPost.Date.Time)
		fmt.Printf("⚠️ Limited analysis to latest %d posts since %s\n", len(posts), since.String())
	}

	spinner := spinner.New(os.Stdout)
	spinner.Start(ctx, "fetching %d reactions on posts")

	var allReactions Reactions
	for i, post := range posts {
		spinner.Progress("checking reactions on posts %d/%d: %d reactions found", i, len(posts), len(allReactions))
		reactions, err := post.FetchReactions(ctx, client, repo)
		if err != nil {
			return err
		}
		allReactions.Append(reactions...)
	}
	spinner.Done("✔️ fetched reactions on %d posts: %d reactions found", len(posts), len(allReactions))

	allReactions.Clean()

	fmt.Println("Stats since", since)
	fmt.Println(len(allPosts), "messages on repository")
	fmt.Println(len(posts), "analyzed messages")
	postsWithReactions := allReactions.Posts()
	fmt.Println(len(postsWithReactions), "messages with reactions")
	fmt.Println()

	if len(postsWithReactions) == 0 {
		return nil
	}

	var reactionDetails []string
	topReactions := allReactions.Reactions()
	for _, reaction := range topReactions {
		reactionDetails = append(reactionDetails, fmt.Sprintf("%d: %s", reaction.Count, reaction.Value))
	}
	fmt.Printf("Total reactions: %d (%s)\n\n", len(allReactions), strings.Join(reactionDetails, " "))

	topPosts := postsWithReactions.Top(5)
	if len(postsWithReactions) > len(topPosts) {
		fmt.Println("Messages with most reactions:")
	} else {
		fmt.Println("Messages with reactions:")
	}

	for _, post := range topPosts {
		fmt.Printf("Reactions:    %d\n", post.Count)
		fmt.Print(post.Value.String())
		fmt.Println()
	}
	fmt.Println()

	authors := allReactions.Authors()
	topAuthors := authors.Top(5)
	if len(authors) > len(topAuthors) {
		fmt.Println("Total users who got reactions:", len(authors))
		fmt.Println("\nTop users who got reactions:")
	} else {
		fmt.Println("Users who got reactions:")
	}

	maxSizeCount := topAuthors.MaxSizeCount()
	maxSizeLogin := topAuthors.MaxSizeValue(func(u github.User) string {
		if u.Login == nil {
			return ""
		}
		return *u.Login
	})

	for _, user := range topAuthors {
		fmt.Printf("%*s %-*s %s\n", maxSizeCount, strconv.Itoa(user.Count), maxSizeLogin, user.Value, user.Value.GitHubURL())
	}
	fmt.Println()

	users := allReactions.Users()
	topUsers := users.Top(5)
	if len(users) > len(topUsers) {
		fmt.Println("Total users who reacted:", len(users))
		fmt.Println("Top users who reacted:")
	} else {
		fmt.Println("Users who reacted:", len(users))
	}

	maxSizeCount = topUsers.MaxSizeCount()
	maxSizeLogin = topUsers.MaxSizeValue(func(u github.User) string {
		if u.Login == nil {
			return ""
		}
		return *u.Login
	})

	for _, user := range topUsers {
		fmt.Printf("%*s %-*s %s\n", maxSizeCount, strconv.Itoa(user.Count), maxSizeLogin, user.Value, user.Value.GitHubURL())
	}
	fmt.Println()

	fmt.Println("Last reactions:")
	for _, reaction := range allReactions {
		fmt.Print(reaction.String())
		fmt.Println()
	}

	return nil
}

type cliOptions struct {
	author string
	limit  int
	since  timeago.RelativeDate
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err := run(ctx)
	switch {
	case errors.Is(err, flag.ErrHelp):
		// nothing to do, the help message has already been displayed
	case errors.Is(err, context.Canceled) && ctx.Err() != nil:
		// handle the CTRL+C case silently
		os.Exit(130) // classic exit code for a SIGINT (Ctrl+C) termination

	case err != nil:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1) // return a non-zero exit code for any other error
	}
}
