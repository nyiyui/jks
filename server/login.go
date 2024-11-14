package server

import (
	"encoding/gob"
	"encoding/json"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

func init() {
	gob.Register(new(githubUserData))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	session, err := s.store.Get(r, "login-oauth2")
	if err != nil {
		http.Error(w, "session failure", 400)
		return
	}
	verifier := oauth2.GenerateVerifier()
	session.Values["verifier"] = verifier
	session.Save(r, w)
	url := s.oauthConfig.AuthCodeURL("", oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, url, 302)
}

type githubUserData struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

func (s *Server) loginCallback(w http.ResponseWriter, r *http.Request) {
	session, err := s.store.Get(r, "login-oauth2")
	if err != nil {
		http.Error(w, "session failure", 400)
		return
	}
	loginSession, err := s.store.Get(r, "login")
	if err != nil {
		http.Error(w, "session failure", 400)
		return
	}

	verifier := session.Values["verifier"].(string)
	delete(session.Values, "verifier")
	code := r.URL.Query().Get("code")
	token, err := s.oauthConfig.Exchange(r.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	client := s.oauthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Error(w, "failed to get user data from GitHub (request)", 500)
		return
	}
	if resp.StatusCode != 200 {
		http.Error(w, "failed to get user data from GitHub (response status code)", 500)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to get user data from GitHub (response body)", 500)
		return
	}
	var data githubUserData
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "failed to get user data from GitHub (response json)", 500)
		return
	}
	loginSession.Values["githubUserData"] = data
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "session failure", 400)
		return
	}
	err = loginSession.Save(r, w)
	if err != nil {
		http.Error(w, "session failure", 400)
		return
	}
}
