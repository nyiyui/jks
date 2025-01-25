package linkdata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"nyiyui.ca/seekback-server/tokens"
)

type LinkProvider interface {
	GetLinks(link *url.URL, ctx context.Context) (LinkData, error)
}

type RemoteLinkProvider struct {
	baseURI *url.URL
	token   tokens.Token
}

func (r *RemoteLinkProvider) GetLinks(link *url.URL, ctx context.Context) (LinkData, error) {
	u := r.baseURI.JoinPath("linkdata")
	u.RawQuery = url.Values{
		"url": {link.String()},
	}.Encode()
	var c http.Client
	c.Timeout = 10 * time.Second
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return LinkData{}, err
	}
	req.Header.Set("X-API-Token", r.token.String())
	resp, err := c.Do(req)
	if err != nil {
		return LinkData{}, err
	}
	var ld LinkData
	err = json.NewDecoder(resp.Body).Decode(&ld)
	if err != nil {
		return LinkData{}, err
	}
	return ld, nil
}
