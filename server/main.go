package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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
	s.mux.HandleFunc("POST /activity/{id}/extend", s.activityExtend)
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
	as, err := s.st.ActivityLatestN(r.Context(), 3)
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	ts := make([]storage.Task, len(as))
	for i, a := range as {
		t, err := s.st.TaskGet(a.TaskID, r.Context())
		if err != nil {
			log.Printf("storage: %s", err)
			http.Error(w, "storage error", 500)
			return
		}
		ts[i] = t
	}
	log.Printf("got: %s", as)

	err = s.tp.ExecuteTemplate(w, "activity-latest.html", map[string]interface{}{
		"latest": as,
		"tasks":  ts,
	})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
}

func (s *Server) activityExtend(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprint(err), 400)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "id must be int", 422)
		return
	}
	timeEndRaw, err := time.ParseInLocation("15:04", r.FormValue("time_end"), time.Local)
	if err != nil {
		http.Error(w, "time_end must be in form 15:04", 422)
		return
	}
	now := time.Now()
	timeEnd := time.Date(now.Year(), now.Month(), now.Day(), timeEndRaw.Hour(), timeEndRaw.Minute(), 0, 0, time.Local)
	log.Printf("time end: %s", timeEnd)
	a, err := s.st.ActivityGet(id, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	a.TimeEnd = timeEnd
	err = s.st.ActivityEdit(a, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	http.Error(w, "done", 200)
}
