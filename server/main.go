package server

import (
	"encoding/json"
	"net/http"

	"nyiyui.ca/jks/database"
	"nyiyui.ca/jks/storage"
)

type Server struct {
	mux *http.ServeMux
	st  storage.Storage
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
}

type ActivityNewQ struct {
	Activity database.Activity
}

func (s *Server) activityNew(w http.ResponseWriter, r *http.Request) {
	var q ActivityNewQ
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, "json decode failed", 422)
		return
	}
	a := q.Activity
	var taskID int64
	if a.TaskID != 0 {
		taskID = a.TaskID
	}
	http.Error(w, "ok", 200)
}
