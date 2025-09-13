package timeago

import (
	"errors"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "Empty",
			input:    "",
			expected: time.Now(),
		},
		{
			name:     "DateOnly",
			input:    "2024-06-01",
			expected: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "RFC3339",
			input:    "2024-06-01T15:04:05Z",
			expected: time.Date(2024, 6, 1, 15, 4, 5, 0, time.UTC),
		},
		{
			name:     "Duration",
			input:    "2h",
			expected: time.Now().Add(-2 * time.Hour),
		},
		{
			name:     "DaysSuffix",
			input:    "3d",
			expected: time.Now().AddDate(0, 0, -3),
		},
		{
			name:     "WeeksSuffix",
			input:    "2w",
			expected: time.Now().AddDate(0, 0, -14),
		},
		{
			name:     "ComplexRelativeDate",
			input:    "3y5w5d3h",
			expected: time.Now().AddDate(-3, 0, -5*7-3),
		},
		{
			name:     "YearsSuffix",
			input:    "2y",
			expected: time.Now().AddDate(-2, 0, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// Allow a few seconds difference for relative dates
			if got.After(time.Now()) || got.Before(tt.expected.Add(-2*time.Second)) {
				t.Errorf("expected around %s, got %s", tt.expected, got)
			}
		})
	}

	tests = []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:  "Wrong order",
			input: "3d3w",
		},
		{
			name:  "Double date d",
			input: "3d3d",
		},
		{
			name:  "Format error",
			input: "3dd",
		},
		{
			name:  "UnsupportedFormat",
			input: "invalid-format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDate(tt.input)
			if err == nil {
				t.Errorf("expected error, got nil")
			} else if !errors.Is(err, ErrUnsupportedDateFormat) {
				t.Errorf("expected error %v, got %v", ErrUnsupportedDateFormat, err)
			}
		})
	}
}
