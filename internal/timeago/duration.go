package timeago

import (
	"fmt"
	"time"
)

func Convert(t time.Time) string {
	return ConvertDuration(time.Since(t))
}

func ConvertDuration(d time.Duration) string {
	if d < 0 {
		return "in the future"
	}
	if d < 2*time.Minute {
		return fmt.Sprintf("%d seconds ago", int(d.Seconds()))
	}
	if d < 2*time.Hour {
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	}

	if d < 49*time.Hour {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	}

	// this is completely wrong in terms of timezone consideration,
	// but it is enough for our needs
	days := int(d.Hours() / 24)

	if days < 22 {
		return fmt.Sprintf("%d days ago", days)
	}

	if days < 31*2 {
		return fmt.Sprintf("%d weeks ago", days/7)
	}

	if days < 365*2 {
		return fmt.Sprintf("%d months ago", days/30)
	}

	return fmt.Sprintf("%d years ago", days/365)
}
