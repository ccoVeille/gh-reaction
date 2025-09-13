package timeago_test

import (
	"testing"
	"time"

	"github.com/ccoVeille/gh-reaction/internal/timeago"
)

func TestConvertDuration(t *testing.T) {
	day := 24 * time.Hour
	month := 30 * day
	year := 365 * day

	cases := []struct {
		input    time.Duration
		expected string
	}{
		{-1 * time.Second, "in the future"},
		{5 * time.Second, "5 seconds ago"},
		{59 * time.Second, "59 seconds ago"},
		{1 * time.Minute, "60 seconds ago"},
		{2 * time.Minute, "2 minutes ago"},
		{10 * time.Minute, "10 minutes ago"},
		{59 * time.Minute, "59 minutes ago"},
		{1 * time.Hour, "60 minutes ago"},
		{2 * time.Hour, "2 hours ago"},
		{23 * time.Hour, "23 hours ago"},
		{24 * time.Hour, "24 hours ago"},
		{48 * time.Hour, "48 hours ago"},
		{49 * time.Hour, "2 days ago"},
		{6 * day, "6 days ago"},
		{20 * day, "20 days ago"},
		{27 * day, "3 weeks ago"},
		{30 * day, "4 weeks ago"},
		{60 * day, "8 weeks ago"},
		{62 * day, "2 months ago"},
		{1 * year, "12 months ago"},
		{18 * month, "18 months ago"},
		{23 * month, "23 months ago"},
		{2 * year, "2 years ago"},
		{100 * year, "100 years ago"},
	}

	for _, c := range cases {
		got := timeago.ConvertDuration(c.input)
		if got != c.expected {
			t.Errorf("Convert(%v) = %q, want %q", c.input, got, c.expected)
		}
	}
}
