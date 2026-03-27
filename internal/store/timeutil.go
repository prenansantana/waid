package store

import (
	"fmt"
	"time"
)

// ParseTime tries RFC3339Nano then the SQLite default TIMESTAMP format.
func ParseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999999",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time format: %q", s)
}
