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
	tps map[string]*template.Template
}

func New(st storage.Storage) (*Server, error) {
	s := &Server{
		mux: http.NewServeMux(),
		st:  st,
	}
	err := s.setup()
	return s, err
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) setup() error {
	s.mux.HandleFunc("GET /activity/{id}", s.activityView)
	s.mux.HandleFunc("GET /task/{id}", s.taskView)
	s.mux.HandleFunc("POST /activity/new", s.activityNew)
	s.mux.HandleFunc("GET /activity/latest", s.activityLatest)
	s.mux.HandleFunc("POST /activity/{id}/extend", s.activityExtend)
	s.mux.HandleFunc("POST /activity/{id}/resume", s.activityResume)
	s.mux.HandleFunc("GET /activity/new-with-task", s.activityNewWithTask)
	s.mux.HandleFunc("GET /day/{date}", s.dayView)
	s.mux.HandleFunc("GET /day/today", s.dayViewToday)
	err := s.parseTemplates()
	return err
}

func (s *Server) activityView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "id must be int", 422)
		return
	}
	a, err := s.st.ActivityGet(id, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	t, err := s.st.TaskGet(a.TaskID, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	err = s.tps["activity.html"].Execute(w, map[string]interface{}{
		"activity": a,
		"task":     t,
	})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
}

func (s *Server) taskView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "id must be int", 422)
		return
	}
	t, err := s.st.TaskGet(id, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	asw, err := s.st.TaskGetActivities(id, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	defer asw.Close()
	as, err := asw.Get(100, 0)
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	if len(as) == 100 {
		http.Error(w, "too many activities", 500)
		return
	}
	var totalSpent time.Duration
	for _, a := range as {
		totalSpent += a.TimeEnd.Sub(a.TimeStart)
	}
	err = s.tps["task.html"].Execute(w, map[string]interface{}{
		"task":       t,
		"activities": as,
		"totalSpent": totalSpent,
	})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
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
	as, err := s.st.ActivityLatestN(r.Context(), 7)
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

	err = s.tps["activity-latest.html"].Execute(w, map[string]interface{}{
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
	timeEnd, err := parseFormTime(r.FormValue("time_end"))
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
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
	http.Redirect(w, r, "/activity/latest", 302)
}

func (s *Server) activityResume(w http.ResponseWriter, r *http.Request) {
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
	location := r.FormValue("location")
	timeEnd, err := parseFormTime(r.FormValue("time_end"))
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	timeStart, err := parseFormTime(r.FormValue("time_start"))
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	a, err := s.st.ActivityGet(id, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	a.ID = 0
	a.TimeStart = timeStart
	a.TimeEnd = timeEnd
	a.Location = location
	err = s.st.ActivityAdd(a, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	http.Redirect(w, r, "/activity/latest", 302)
}

func (s *Server) activityNewWithTask(w http.ResponseWriter, r *http.Request) {
	err := s.tps["activity-new-with-task.html"].Execute(w, map[string]interface{}{})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
}

func (s *Server) dayView(w http.ResponseWriter, r *http.Request) {
	date, err := time.ParseInLocation("2006-01-02", r.PathValue("date"), time.Local)
	if err != nil {
		http.Error(w, "invalid date format", 422)
		return
	}
	dateEnd := date.Add(24 * time.Hour)
	asw, err := s.st.ActivityRange(date, dateEnd, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	defer asw.Close()
	as, err := asw.Get(100, 0)
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	if len(as) == 100 {
		http.Error(w, "too many activities", 500)
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
	err = s.tps["day.html"].Execute(w, map[string]interface{}{
		"date":       date,
		"activities": as,
		"tasks":      ts,
	})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
}

func (s *Server) dayViewToday(w http.ResponseWriter, r *http.Request) {
	date := time.Now().Format("2006-01-02")
	http.Redirect(w, r, fmt.Sprintf("/day/%s", date), 302)
}
