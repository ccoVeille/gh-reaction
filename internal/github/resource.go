package github

import (
	"strings"
	"time"

	"github.com/google/go-github/v74/github"

	"github.com/ccoVeille/gh-reaction/internal/timeago"
)

type User struct {
	github.User
}

func (u User) GitHubURL() string {
	if u.Login == nil {
		return ""
	}
	return "https://github.com/" + *u.Login
}

func (u User) IsBot() bool {
	if u.Login == nil {
		return false
	}

	switch strings.ToLower(*u.Login) {
	case
		"coderabbitai[bot]",
		"dependabot[bot]",
		"github-actions[bot]",
		"renovate[bot]", // renovate is the old name for mend
		"mend[bot]",     // mend is the new name for renovate
		"codecov-commenter":
		return true
	}

	return false
}

func (u User) String() string {
	if u.Login == nil {
		return "unknown"
	}

	if u.Name == nil || *u.Name == "" || *u.Login == *u.Name {
		return *u.Login
	}

	return *u.Name + " (" + *u.Login + ")"
}

type Time struct {
	time.Time
}

func (d Time) String() string {
	if d.IsZero() {
		return "forever"
	}
	return timeago.Convert(d.Time)
}

type Reaction struct {
	User      User   `json:"user"`
	Content   string `json:"content"`
	CreatedAt Time   `json:"created_at"`
}

func (r Reaction) Type() string {
	switch r.Content {
	case "+1":
		return "👍"
	case "-1":
		return "👎"
	case "eyes":
		return "👀"
	case "heart":
		return "❤️"
	case "laugh":
		return "😂"
	case "hooray":
		return "🙌"
	case "confused":
		return "😕"
	case "rocket":
		return "🚀"
	default:
		return "🤷" + " unknown reaction " + r.Content
	}
}
