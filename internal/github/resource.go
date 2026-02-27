package github

import (
	"strings"
	"time"

	"github.com/ccoVeille/gh-reaction/internal/timeago"
)

// User wraps github.User to provide additional methods.
type User struct {
	Login    string `json:"login,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"__typename,omitempty"`
	IsViewer bool   `json:"is_viewer,omitempty"`
}

// GitHubURL returns the URL to the user's GitHub profile.
func (u User) GitHubURL() string {
	return "https://github.com/" + u.Login
}

// IsBot reports whether the user is a bot account.
func (u User) IsBot() bool {
	// TODO: consider using this field when available
	// if u.Type == strings.ToLower("Bot") {
	// 	return true
	// }

	switch strings.ToLower(u.Login) {
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
	if u.Login == "" {
		return "unknown"
	}

	if u.Name == "" || u.Login == u.Name {
		return u.Login
	}

	return u.Login + " (" + u.Name + ")"
}

func (u User) PossessiveLabel() string {
	if u.IsViewer {
		return "your"
	}
	return u.String() + "'s"
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
	switch strings.ToLower(r.Content) {
	case
		"+1",        // REST API representation
		"thumbs_up": // GraphQL API representation
		return "👍"
	case
		"-1",          // REST API representation
		"thumbs_down": // GraphQL API representation
		return "👎"
	case "eyes":
		return "👀"
	case "heart":
		return "❤️"
	case "laugh":
		return "😂"
	case "hooray":
		return "🎉"
	case "confused":
		return "😕"
	case "rocket":
		return "🚀"
	default:
		return "🤷" + " unknown reaction " + r.Content
	}
}
