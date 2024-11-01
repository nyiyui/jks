package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/safehtml/template"

	"nyiyui.ca/jks/storage"
)

type Server struct {
	mux *http.ServeMux
	st  storage.Storage
	tp  *template.Template
}

func New(st storage.Storage) (*Server, error) {
	s := &Server{
		mux: http.NewServeMux(),
		st:  st,
	}
	s.setup()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) setup() {
	s.mux.HandleFunc("POST /activity/new", s.activityNew)
	s.mux.HandleFunc("GET /activity/latest", s.activityLatest)
	s.parseTemplates()
}

type ActivityNewQ struct {
	Activity storage.Activity
}

func (s *Server) activityNew(w http.ResponseWriter, r *http.Request) {
	var q ActivityNewQ
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, "json decode failed", 422)
		return
	}
	err = s.st.ActivityAdd(q.Activity, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	http.Error(w, "ok", 200)
}

func (s *Server) activityLatest(w http.ResponseWriter, r *http.Request) {
	a, err := s.st.ActivityLatest(r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	log.Printf("got: %s", a)
	err = s.tp.ExecuteTemplate(w, "activity-latest.html", map[string]interface{}{
		"latest": a,
	})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
}
