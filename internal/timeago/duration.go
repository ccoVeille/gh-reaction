package timeago

import (
	"fmt"
	"time"
)

// Convert converts a [time.Time] into a human-readable relative time string.
func Convert(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		return "in the future"
	}

	return ConvertDuration(time.Since(t)) + " ago"
}

// ConvertDuration converts a [time.Duration] into a human-readable relative time string.
func ConvertDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	if d < 2*time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < 2*time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}

	if d < 49*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}

	// this is completely wrong in terms of timezone consideration,
	// but it is enough for our needs
	days := int(d.Hours() / 24)

	if days < 22 {
		return fmt.Sprintf("%d days", days)
	}

	if days < 31*2 {
		return fmt.Sprintf("%d weeks", days/7)
	}

	if days < 365*2 {
		return fmt.Sprintf("%d months", days/30)
	}

	return fmt.Sprintf("%d years", days/365)
}
