package app

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ccoVeille/gh-reaction/internal/github"
	"github.com/ccoVeille/gh-reaction/internal/timeago"
)

type PostType string

const (
	PostTypeIssue        PostType = "issue"
	PostTypePullRequest  PostType = "pull_request"
	PostTypeRelease      PostType = "release"
	PostTypeComment      PostType = "comment"
	PostTypeIssueComment PostType = "issue_comment"
	PostTypePRComment    PostType = "pull_request_comment"
)

func (pt PostType) String() string {
	switch pt {
	case "pull_request_comment":
		return "PR comment"
	case "issue_comment":
		return "issue comment"
	case PostTypePullRequest:
		return "pull request"
	case PostTypeIssue:
		return "issue"
	case PostTypeRelease:
		return "release"
	case PostTypeComment:
		return "comment"
	default:
		return strings.ReplaceAll(string(pt), "_", " ")
	}
}

type Post struct {
	Type    PostType
	Date    github.Time
	Content string
	Author  github.User
	Link    string
	ID      string
}

func (p Post) ContentPreview() string {
	content := p.Content
	for _, l := range strings.Split(content, "\n") {
		if strings.HasPrefix(l, ">") {
			continue
		}

		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}

		content = l
		break
	}

	const maxLen = 100
	if content == p.Content {
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

type Reactions []ReactionTo

func (r *Reactions) Clean() {
	clean := slices.DeleteFunc(*r, func(r1 ReactionTo) bool {
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

	slices.SortFunc(v, func(a, b ValueCount[T]) int {
		if a.Count == b.Count {
			return cmp.Compare(fmt.Sprint(a.Value), fmt.Sprint(b.Value))
		}

		return b.Count - a.Count
	})

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
		key := reaction.Reaction.User.Login
		u, found := userCounts[key]
		if !found {
			u = ValueCount[github.User]{Value: reaction.Reaction.User}
		}
		u.Count++
		userCounts[key] = u
	}

	return slices.Collect(maps.Values(userCounts))
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

func formatLocalDateTime(t time.Time) string {
	return t.In(time.Local).Format("2006-01-02 15:04:05 MST")
}

func (r ReactionTo) String() string {
	sb := strings.Builder{}

	reactedAtLocal := formatLocalDateTime(r.Reaction.CreatedAt.Time)
	reactedAtRelative := r.Reaction.CreatedAt.String()
	diff := r.Reaction.CreatedAt.Sub(r.Post.Date.Time)

	fmt.Fprintf(&sb, "  %s %s • %s (%s) • %s after post • %s\n",
		r.Reaction.Type(),
		r.Reaction.User,
		reactedAtLocal,
		reactedAtRelative,
		timeago.ConvertDuration(diff),
		r.Reaction.User.GitHubURL(),
	)

	return sb.String()
}

func (p Post) String() string {
	sb := strings.Builder{}

	fmt.Fprintf(&sb, "> %s\n", p.ContentPreview())
	fmt.Fprintf(&sb, "%s %s posted %s (%s)\n%s\n",
		p.Author.PossessiveLabel(),
		p.Type,
		formatLocalDateTime(p.Date.Time),
		p.Date.String(),
		p.Link,
	)

	return sb.String()
}
