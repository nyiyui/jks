package server

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

type key struct{}

var LoginUserDataKey key

func init() {
	gob.RegisterName("githubUserData", githubUserData{})
}

func (s *Server) mainLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginSession, err := s.store.Get(r, "login")
		if err != nil {
			log.Printf("login session get: %s", err)
			http.Error(w, "session failure", 400)
			return
		}
		_, ok := loginSession.Values["githubUserData"]
		if !ok {
			http.Redirect(w, r, "/login", 302)
			return
		}
		data := loginSession.Values["githubUserData"].(githubUserData)
		if data.Login != s.mainUser {
			http.Error(w, "must be main user", 401)
			return
		}
		r2 := r.WithContext(context.WithValue(r.Context(), LoginUserDataKey, data))
		next.ServeHTTP(w, r2)
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Has("code") {
		http.Error(w,"login page should not have query parameter `code' - make sure your redirect URI is set correctly.", 500)
		return
	}
	session, err := s.store.Get(r, "login-oauth2")
	if err != nil {
		log.Printf("session get: %s", err)
		http.Error(w, "session failure", 400)
		return
	}
	verifier := oauth2.GenerateVerifier()
	session.Values["verifier"] = verifier
	err = session.Save(r, w)
	if err != nil {
		log.Printf("session save: %s", err)
		http.Error(w, "session failure", 400)
		return
	}
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
		log.Printf("session get: %s", err)
		http.Error(w, "session failure", 400)
		return
	}
	loginSession, err := s.store.Get(r, "login")
	if err != nil {
		log.Printf("login session get: %s", err)
		http.Error(w, "session failure", 400)
		return
	}

	log.Printf("session: %s", session.Values)

	_, ok := session.Values["verifier"]
	if !ok {
		http.Error(w, "try logging in again", 400)
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
		log.Printf("session save: %s", err)
		http.Error(w, "session failure", 400)
		return
	}
	err = loginSession.Save(r, w)
	if err != nil {
		log.Printf("login session save: %s", err)
		http.Error(w, "session failure", 400)
		return
	}
	http.Error(w, fmt.Sprintf("logged in as %s", data.Login), 200)
}
