package server

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"

	"github.com/google/safehtml/template"

	"nyiyui.ca/jks/layout"
	"nyiyui.ca/jks/storage"
)

func composeFunc(handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) http.Handler {
	var h http.Handler = handler
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

type Server struct {
	mux         *http.ServeMux
	st          storage.Storage
	tps         map[string]*template.Template
	oauthConfig *oauth2.Config
	store       sessions.Store
	mainUser    string
}

func New(st storage.Storage, oauthConfig *oauth2.Config, store sessions.Store, adminUser string) (*Server, error) {
	s := &Server{
		mux:         http.NewServeMux(),
		st:          st,
		oauthConfig: oauthConfig,
		store:       store,
		mainUser:    adminUser,
	}
	err := s.setup()
	return s, err
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) setup() error {
	s.mux.HandleFunc("GET /login", s.login)
	s.mux.HandleFunc("GET /login/callback", s.loginCallback)

	s.mux.Handle("GET /undone-tasks", composeFunc(s.undoneTasks, s.mainLogin))
	s.mux.Handle("GET /activity/{id}", composeFunc(s.activityView, s.mainLogin))
	s.mux.Handle("GET /task/new", composeFunc(s.taskNew, s.mainLogin))
	s.mux.Handle("POST /task/new", composeFunc(s.taskNewPost, s.mainLogin))
	s.mux.Handle("GET /task/{id}", composeFunc(s.taskView, s.mainLogin))
	s.mux.Handle("GET /task/{id}/activity/new", composeFunc(s.taskActivityNew, s.mainLogin))
	s.mux.Handle("POST /task/{id}/activity/new", composeFunc(s.taskActivityNewPost, s.mainLogin))
	s.mux.Handle("GET /task/new/activity/new", composeFunc(s.taskNewActivityNew, s.mainLogin))
	s.mux.Handle("POST /task/new/activity/new", composeFunc(s.taskNewActivityNewPost, s.mainLogin))
	s.mux.Handle("GET /task/{id}/plan/new", composeFunc(s.taskPlanNew, s.mainLogin))
	s.mux.Handle("POST /task/{id}/plan/new", composeFunc(s.taskPlanNewPost, s.mainLogin))
	s.mux.Handle("GET /activity/latest", composeFunc(s.activityLatest, s.mainLogin))
	s.mux.Handle("POST /activity/{id}/extend", composeFunc(s.activityExtend, s.mainLogin))
	s.mux.Handle("POST /activity/{id}/resume", composeFunc(s.activityResume, s.mainLogin))
	s.mux.Handle("GET /day/{date}", composeFunc(s.dayView, s.mainLogin))
	s.mux.Handle("GET /day/yesterday", composeFunc(s.makeDayViewDelta(-1), s.mainLogin))
	s.mux.Handle("GET /day/today", composeFunc(s.makeDayViewDelta(0), s.mainLogin))
	s.mux.Handle("GET /day/tomorrow", composeFunc(s.makeDayViewDelta(1), s.mainLogin))
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
	s.renderTemplate("undone-tasks.html", w, r, map[string]interface{}{
		"tasks": ts,
	})
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
	s.renderTemplate("activity.html", w, r, map[string]interface{}{
		"activity": a,
		"task":     t,
	})
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
	s.renderTemplate("task.html", w, r, map[string]interface{}{
		"task":       t,
		"activities": as,
		"totalSpent": totalSpent,
	})
	return
}

func (s *Server) taskNew(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate("task-new.html", w, r, map[string]interface{}{})
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

func (s *Server) taskNewActivityNew(w http.ResponseWriter, r *http.Request) {
	preset := map[string]string{}
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			preset[k] = v[0]
		}
	}
	if _, ok := preset["Deadline"]; !ok {
		preset["Deadline"] = time.Now().Format("2006-01-02T15:04")
	}
	if _, ok := preset["Due"]; !ok {
		preset["Due"] = time.Now().Format("2006-01-02T15:04")
	}
	if _, ok := preset["TimeStart"]; !ok {
		preset["TimeStart"] = time.Now().Format("2006-01-02T15:04")
	}
	if _, ok := preset["TimeEnd"]; !ok {
		preset["TimeEnd"] = time.Now().Format("2006-01-02T15:04")
	}
	s.renderTemplate("task-new-activity-new.html", w, r, map[string]interface{}{"preset": preset})
	return
}

type taskNewActivityNewQ struct {
	storage.Task
	storage.Activity
}

func (s *Server) taskNewActivityNewPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "parsing form data failed", 400)
		return
	}

	var parsed taskNewActivityNewQ
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
	taskID, err := s.st.TaskAdd(parsed.Task, r.Context())
	if err != nil {
		log.Printf("storage: add task: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	parsed.Activity.TaskID = taskID
	activityID, err := s.st.ActivityAdd(parsed.Activity, r.Context())
	if err != nil {
		log.Printf("storage: add activity: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/activity/%d", activityID), 302)
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
	s.renderTemplate("task-activity-new.html", w, r, map[string]interface{}{
		"task":         t,
		"plans":        plans,
		"selectedPlan": selectedPlan,
	})
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
	s.renderTemplate("task-plan-new.html", w, r, map[string]interface{}{
		"task": t,
	})
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

	s.renderTemplate("activity-latest.html", w, r, map[string]interface{}{
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

	events := make([]layout.Box, len(as)+len(ps))
	for i, a := range as {
		events[i] = a
	}
	for i, p := range ps {
		events[len(as)+i] = p
	}

	nColumns, columns := layout.Layout(events)

	s.renderTemplate("day.html", w, r, map[string]interface{}{
		"date":     date,
		"events":   events,
		"tasks":    tasksByID,
		"nColumns": nColumns,
		"columns":  columns,
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
