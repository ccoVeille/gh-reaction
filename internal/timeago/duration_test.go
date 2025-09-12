package timeago_test

import (
	"testing"
	"time"

	"github.com/ccoVeille/gh-reaction/internal/timeago"
)

func TestConvertDuration(t *testing.T) {
	cases := []struct {
		input    time.Duration
		expected string
	}{
		{-1 * time.Second, "in the future"},
		{5 * time.Second, "5 seconds ago"},
		{59 * time.Second, "59 seconds ago"},
		{1 * time.Minute, "1 minutes ago"},
		{10 * time.Minute, "10 minutes ago"},
		{59 * time.Minute, "59 minutes ago"},
		{1 * time.Hour, "1 hours ago"},
		{23 * time.Hour, "23 hours ago"},
		{24 * time.Hour, "1 days ago"},
		{6 * 24 * time.Hour, "6 days ago"},
		{7 * 24 * time.Hour, "1 weeks ago"},
		{29 * 24 * time.Hour, "4 weeks ago"},
		{30 * 24 * time.Hour, "1 months ago"},
		{364 * 24 * time.Hour, "12 months ago"},
		{18 * 30 * 24 * time.Hour, "18 months ago"},
		{2 * 365 * 24 * time.Hour, "2 years ago"},
	}

	for _, c := range cases {
		got := timeago.ConvertDuration(c.input)
		if got != c.expected {
			t.Errorf("Convert(%v) = %q, want %q", c.input, got, c.expected)
		}
	}
}
