package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"nyiyui.ca/seekback-server/storage"
)

type Event interface {
	TimeRange() (start, end time.Time)
}

const QueryLayout = "2006-01-02T15:04:05"

func (s *Server) getEvents(start, end time.Time, ctx context.Context) ([]Event, error) {
	u := s.seekbackServerBaseURI.JoinPath("events")

	u.RawQuery = url.Values{
		"time_start": {start.Format(time.RFC3339)},
		"time_end":   {end.Format(time.RFC3339)},
		"overlap":    {"true"},
	}.Encode()
	var c http.Client
	c.Timeout = 10 * time.Second
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Token", s.seekbackServerToken.String())
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	var events []storage.SamplePreview
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		return nil, err
	}
	result := make([]Event, len(events))
	for i := range events {
		result[i] = events[i]
	}
	return result, nil
}
