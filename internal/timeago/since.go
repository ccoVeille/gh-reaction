package timeago

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

// RelativeDate is a wrapper around [time.Time] that provides a custom string representation.
type RelativeDate struct {
	time.Time
}

// NewRelativeDate creates a new RelativeDate instance.
func NewRelativeDate(t time.Time) RelativeDate {
	return RelativeDate{Time: t}
}

// String returns the string representation of the RelativeDate.
//
// It satisfies the [flag.Value] and [fmt.Stringer] interface.
func (r RelativeDate) String() string {
	return fmt.Sprintf("%v (%v)", r.Time.Format(time.RFC3339), Convert(r.Time))
}

// Set sets the RelativeDate from a string value.
//
// It satisfies the [flag.Value] interface.
func (r *RelativeDate) Set(value string) error {
	t, err := parseDate(value)
	if err != nil {
		return err
	}
	r.Time = t
	return nil
}

var _ flag.Value = (*RelativeDate)(nil)

// ErrUnsupportedDateFormat is returned when the provided date format is not supported.
var ErrUnsupportedDateFormat = errors.New("unsupported date format")

func parseDate(analyzed string) (time.Time, error) {
	// Date only requires to set the location to UTC
	t, err := time.ParseInLocation(time.DateOnly, analyzed, time.UTC)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse(time.RFC3339, analyzed)
	if err == nil {
		return t, nil
	}

	var (
		offsetYear, offsetDay int
		unit                  int
		remaining             string
	)

	if strings.Contains(analyzed, "y") {
		_, err = fmt.Sscanf(analyzed, "%dy%s", &unit, &analyzed)
		if err != nil && !errors.Is(err, io.EOF) {
			return time.Time{}, ErrUnsupportedDateFormat
		}
		if strings.Contains(remaining, "y") {
			return time.Time{}, ErrUnsupportedDateFormat
		}

		offsetYear = unit
		analyzed, remaining = remaining, ""
	}

	if strings.Contains(analyzed, "w") {
		_, err = fmt.Sscanf(analyzed, "%dw%s", &unit, &remaining)
		if err != nil && !errors.Is(err, io.EOF) {
			return time.Time{}, ErrUnsupportedDateFormat
		}
		if strings.Contains(remaining, "w") {
			return time.Time{}, ErrUnsupportedDateFormat
		}
		offsetDay += 7 * unit
		analyzed, remaining = remaining, ""
	}

	if strings.Contains(analyzed, "d") {
		_, err = fmt.Sscanf(analyzed, "%dd%s", &unit, &remaining)
		if err != nil && !errors.Is(err, io.EOF) {
			return time.Time{}, ErrUnsupportedDateFormat
		}
		if strings.Contains(remaining, "d") {
			return time.Time{}, ErrUnsupportedDateFormat
		}
		offsetDay += unit
		analyzed, remaining = remaining, ""
	}

	t = time.Now().AddDate(-offsetYear, 0, -offsetDay)
	if analyzed == "" {
		return t, nil
	}

	dur, err := time.ParseDuration(analyzed)
	if err != nil {
		return time.Time{}, ErrUnsupportedDateFormat
	}
	return t.Add(-dur), nil
}
