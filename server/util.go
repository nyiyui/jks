package server

import (
	"errors"
	"time"
)

// parseFormTime parses the HTML <input type="time" /> form value.
func parseFormTime(s string) (time.Time, error) {
	timeRaw, err := time.ParseInLocation("15:04", s, time.Local)
	if err != nil {
		return time.Time{}, errors.New("time must be in form 15:04")
	}
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), timeRaw.Hour(), timeRaw.Minute(), 0, 0, time.Local), nil
}
