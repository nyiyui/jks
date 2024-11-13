package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gorilla/schema"

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
	s.mux.HandleFunc("GET /undone-tasks", s.undoneTasks)
	s.mux.HandleFunc("GET /activity/{id}", s.activityView)
	s.mux.HandleFunc("GET /task/new", s.taskNew)
	s.mux.HandleFunc("POST /task/new", s.taskNewPost)
	s.mux.HandleFunc("GET /task/{id}", s.taskView)
	s.mux.HandleFunc("GET /task/{id}/activity/new", s.taskActivityNew)
	s.mux.HandleFunc("POST /task/{id}/activity/new", s.taskActivityNewPost)
	s.mux.HandleFunc("GET /task/{id}/plan/new", s.taskPlanNew)
	s.mux.HandleFunc("POST /task/{id}/plan/new", s.taskPlanNewPost)
	s.mux.HandleFunc("POST /activity/new", s.activityNew)
	s.mux.HandleFunc("GET /activity/latest", s.activityLatest)
	s.mux.HandleFunc("POST /activity/{id}/extend", s.activityExtend)
	s.mux.HandleFunc("POST /activity/{id}/resume", s.activityResume)
	s.mux.HandleFunc("GET /day/{date}", s.dayView)
	s.mux.HandleFunc("GET /day/yesterday", s.makeDayViewDelta(-1))
	s.mux.HandleFunc("GET /day/today", s.makeDayViewDelta(0))
	s.mux.HandleFunc("GET /day/tomorrow", s.makeDayViewDelta(1))
	err := s.parseTemplates()
	return err
}

func (s *Server) undoneTasks(w http.ResponseWriter, r *http.Request) {
	tsw, err := s.st.TaskSearch("", time.Now(), r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	ts, err := tsw.Get(100, 0)
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	if len(ts) == 100 {
		http.Error(w, "too many activities", 500)
		return
	}
	err = s.tps["undone-tasks.html"].Execute(w, map[string]interface{}{
		"tasks": ts,
	})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
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

func (s *Server) taskNew(w http.ResponseWriter, r *http.Request) {
	err := s.tps["task-new.html"].Execute(w, map[string]interface{}{})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
}

func (s *Server) taskNewPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "parsing form data failed", 400)
		return
	}

	var parsed storage.Task
	decoder := schema.NewDecoder()
	decoder.RegisterConverter(time.Time{}, func(s string) reflect.Value {
		t, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local)
		if err != nil {
			return reflect.ValueOf(time.Now())
		}
		return reflect.ValueOf(t)
	})
	err = decoder.Decode(&parsed, r.PostForm)
	if err != nil {
		http.Error(w, fmt.Sprintf("form data decode failed: %s", err), 422)
		return
	}
	taskID, err := s.st.TaskAdd(parsed, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/task/%d", taskID), 302)
}

func (s *Server) taskActivityNew(w http.ResponseWriter, r *http.Request) {
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
	plans, err := s.st.TaskGetPlans(id, 100, 0, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	var selectedPlan int
	for i, plan := range plans {
		if plan.TimeAtAfter.Before(time.Now()) && plan.TimeBefore.After(time.Now()) {
			selectedPlan = i
		}
	}
	err = s.tps["task-activity-new.html"].Execute(w, map[string]interface{}{
		"task":         t,
		"plans":        plans,
		"selectedPlan": selectedPlan,
	})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
}

func (s *Server) taskActivityNewPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "parsing form data failed", 400)
		return
	}

	var a storage.Activity
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
	decoder := schema.NewDecoder()
	decoder.RegisterConverter(time.Time{}, func(s string) reflect.Value {
		t, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local)
		if err != nil {
			return reflect.ValueOf(time.Now())
		}
		return reflect.ValueOf(t)
	})

	planID := r.PostForm.Get("PlanID")
	delete(r.PostForm, "PlanID")

	err = decoder.Decode(&a, r.PostForm)
	if err != nil {
		http.Error(w, fmt.Sprintf("form data decode failed: %s", err), 422)
		return
	}
	if t.Deadline != nil && a.TimeStart.After(*t.Deadline) {
		http.Error(w, "start time cannot be after deadline", 422)
		return
	}
	if t.Deadline != nil && a.TimeEnd.After(*t.Deadline) {
		http.Error(w, "end time cannot be after deadline", 422)
		return
	}
	a.TaskID = id
	activityID, err := s.st.ActivityAdd(a, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	if planID != "" {
		log.Printf("update plan")
		planID2, err := strconv.ParseInt(planID, 10, 64)
		if err != nil {
			http.Error(w, "PlanID must be int or \"\"", 422)
			return
		}
		plan, err := s.st.PlanGet(planID2, r.Context())
		if err != nil {
			log.Printf("storage: %s", err)
			http.Error(w, "storage error", 500)
			return
		}
		plan.ActivityID = activityID
		err = s.st.PlanEdit(plan, r.Context())
		if err != nil {
			log.Printf("storage: %s", err)
			http.Error(w, "storage error", 500)
			return
		}
	}
	http.Redirect(w, r, fmt.Sprintf("/activity/%d", activityID), 302)
}

func (s *Server) taskPlanNew(w http.ResponseWriter, r *http.Request) {
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
	err = s.tps["task-plan-new.html"].Execute(w, map[string]interface{}{
		"task": t,
	})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
}

func (s *Server) taskPlanNewPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "parsing form data failed", 400)
		return
	}

	var p storage.Plan
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
	decoder := schema.NewDecoder()
	decoder.RegisterConverter(time.Time{}, func(s string) reflect.Value {
		t, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local)
		if err != nil {
			return reflect.ValueOf(time.Now())
		}
		return reflect.ValueOf(t)
	})
	decoder.RegisterConverter(time.Duration(0), func(s string) reflect.Value {
		d, err := time.ParseDuration(s)
		if err != nil {
			return reflect.ValueOf(time.Duration(0))
		}
		return reflect.ValueOf(d)
	})
	err = decoder.Decode(&p, r.PostForm)
	if err != nil {
		http.Error(w, fmt.Sprintf("form data decode failed: %s", err), 422)
		return
	}
	if t.Deadline != nil && p.TimeAtAfter.After(*t.Deadline) {
		http.Error(w, "start time cannot be after deadline", 422)
		return
	}
	if t.Deadline != nil && p.TimeBefore.After(*t.Deadline) {
		http.Error(w, "end time cannot be after deadline", 422)
		return
	}
	p.TaskID = id
	_, err = s.st.PlanAdd(p, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/task/%d", id), 302)
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
	_, err = s.st.ActivityAdd(q.Activity, r.Context())
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
	_, err = s.st.ActivityAdd(a, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	http.Redirect(w, r, "/activity/latest", 302)
}

func (s *Server) dayView(w http.ResponseWriter, r *http.Request) {
	date, err := time.ParseInLocation("2006-01-02", r.PathValue("date"), time.Local)
	if err != nil {
		http.Error(w, "invalid date format", 422)
		return
	}
	dateEnd := date.Add(24 * time.Hour)

	ts, as, ps, err := s.st.Range(date, dateEnd, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}

	tasksByID := make(map[int64]storage.Task)
	for _, t := range ts {
		tasksByID[t.ID] = t
	}

	err = s.tps["day.html"].Execute(w, map[string]interface{}{
		"date":       date,
		"activities": as,
		"tasks":      tasksByID,
		"plans":      ps,
	})
	if err != nil {
		log.Printf("template: %s", err)
		http.Error(w, "template error", 500)
		return
	}
	return
}

func (s *Server) makeDayViewDelta(days int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := time.Now().AddDate(0, 0, days).Format("2006-01-02")
		http.Redirect(w, r, fmt.Sprintf("/day/%s", date), 302)
	}
}
