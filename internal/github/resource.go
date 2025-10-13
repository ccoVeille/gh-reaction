package github

import (
	"strings"
	"time"

	"github.com/google/go-github/v74/github"

	"github.com/ccoVeille/gh-reaction/internal/timeago"
)

// User wraps github.User to provide additional methods.
type User struct {
	github.User
}

// GitHubURL returns the URL to the user's GitHub profile.
func (u User) GitHubURL() string {
	if u.Login == nil {
		return ""
	}
	return "https://github.com/" + *u.Login
}

// IsBot reports whether the user is a bot account.
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

// Time wraps time.Time to provide a custom String method.
type Time struct {
	time.Time
}

// String formats the Time in a human-readable relative format.
//
// It implements the [fmt.Stringer] interface.
func (d Time) String() string {
	if d.IsZero() {
		return "forever"
	}
	return timeago.Convert(d.Time)
}

// Reaction wraps github.Reaction to provide additional methods.
type Reaction struct {
	User      User   `json:"user"`
	Content   string `json:"content"`
	CreatedAt Time   `json:"created_at"`
}

// Type returns a string representation of the reaction type.
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
