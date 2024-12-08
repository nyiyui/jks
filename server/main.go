package server

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/deiu/rdf2go"
	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"

	"github.com/google/safehtml/template"

	"nyiyui.ca/jks/layout"
	"nyiyui.ca/jks/rdf"
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
	serializer  *rdf.Serializer
}

func newDecoder(r *http.Request) *schema.Decoder {
	decoder := schema.NewDecoder()
	decoder.RegisterConverter(time.Time{}, func(s string) reflect.Value {
		loc := getTimeLocation(r)
		t, err := time.ParseInLocation("2006-01-02T15:04", s, loc)
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
	return decoder
}

func New(st storage.Storage, oauthConfig *oauth2.Config, store sessions.Store, adminUser string, serializer *rdf.Serializer) (*Server, error) {
	s := &Server{
		mux:         http.NewServeMux(),
		st:          st,
		oauthConfig: oauthConfig,
		store:       store,
		mainUser:    adminUser,
		serializer:  serializer,
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
	s.mux.Handle("GET /login/settings", composeFunc(s.loginSettings, s.mainLogin))
	s.mux.Handle("POST /login/settings", composeFunc(s.loginSettings, s.mainLogin))

	s.mux.Handle("GET /rdf/all", composeFunc(s.getRDF, s.mainLogin))

	s.mux.Handle("GET /undone-tasks", composeFunc(s.undoneTasks, s.mainLogin))
	s.mux.Handle("GET /activity/{id}", composeFunc(s.activityView, s.mainLogin))
	s.mux.Handle("GET /activity/{id}/edit", composeFunc(s.activityEdit, s.mainLogin))
	s.mux.Handle("POST /activity/{id}/edit", composeFunc(s.activityEditPost, s.mainLogin))
	s.mux.Handle("GET /task/new", composeFunc(s.taskNew, s.mainLogin))
	s.mux.Handle("POST /task/new", composeFunc(s.taskNewPost, s.mainLogin))
	s.mux.Handle("GET /task/{id}", composeFunc(s.taskView, s.mainLogin))
	s.mux.Handle("GET /task/{id}/edit", composeFunc(s.taskEdit, s.mainLogin))
	s.mux.Handle("POST /task/{id}/edit", composeFunc(s.taskEditPost, s.mainLogin))
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

func (s *Server) activityEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "id must be int", 422)
		return
	}
	a, err := s.st.ActivityGet(id, r.Context())
	if err != nil {
		log.Printf("storage: activity get: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	t, err := s.st.TaskGet(a.TaskID, r.Context())
	if err != nil {
		log.Printf("storage: task get: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	s.renderTemplate("activity-edit.html", w, r, map[string]interface{}{
		"activity": a,
		"task":     t,
	})
	return
}

func (s *Server) activityEditPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "id must be int", 422)
		return
	}
	err = r.ParseForm()
	if err != nil {
		http.Error(w, "parsing form data failed", 400)
		return
	}
	a, err := s.st.ActivityGet(id, r.Context())
	if err != nil {
		log.Printf("storage: activity get: %s", err)
		http.Error(w, "storage error", 500)
		return
	}

	decoder := newDecoder(r)
	var parsed storage.Activity
	err = decoder.Decode(&parsed, r.PostForm)
	if err != nil {
		http.Error(w, fmt.Sprintf("form data decode failed: %s", err), 422)
		return
	}
	parsed.ID = id
	parsed.TaskID = a.TaskID // do not allow changing task ID
	err = s.st.ActivityEdit(parsed, r.Context())
	if err != nil {
		log.Printf("storage: activity edit: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/activity/%d", id), 302)
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
	ps, err := s.st.TaskGetPlans(id, 100, 0, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	if len(ps) == 100 {
		http.Error(w, "too many plans", 500)
		return
	}
	var totalSpent time.Duration
	for _, a := range as {
		totalSpent += a.TimeEnd.Sub(a.TimeStart)
	}
	s.renderTemplate("task.html", w, r, map[string]interface{}{
		"task":       t,
		"activities": as,
		"plans":      ps,
		"totalSpent": totalSpent,
	})
	return
}

func (s *Server) taskEdit(w http.ResponseWriter, r *http.Request) {
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
	s.renderTemplate("task-edit.html", w, r, map[string]interface{}{
		"task":       t,
		"activities": as,
		"totalSpent": totalSpent,
	})
	return
}

func (s *Server) taskEditPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "id must be int", 422)
		return
	}

	err = r.ParseForm()
	if err != nil {
		http.Error(w, "parsing form data failed", 400)
		return
	}

	decoder := newDecoder(r)
	var parsed storage.Task
	err = decoder.Decode(&parsed, r.PostForm)
	if err != nil {
		http.Error(w, fmt.Sprintf("form data decode failed: %s", err), 422)
		return
	}
	parsed.ID = id
	err = s.st.TaskEdit(parsed, r.Context())
	if err != nil {
		log.Printf("storage: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/task/%d", id), 302)
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

	decoder := newDecoder(r)
	var parsed storage.Task
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
	loc := getTimeLocation(r)
	preset := map[string]string{}
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			preset[k] = v[0]
		}
	}
	now := time.Now().In(loc)
	if _, ok := preset["Deadline"]; !ok {
		preset["Deadline"] = now.Format("2006-01-02T15:04")
	}
	if _, ok := preset["Due"]; !ok {
		preset["Due"] = now.Format("2006-01-02T15:04")
	}
	if _, ok := preset["TimeStart"]; !ok {
		preset["TimeStart"] = now.Format("2006-01-02T15:04")
	}
	if _, ok := preset["TimeEnd"]; !ok {
		preset["TimeEnd"] = now.Format("2006-01-02T15:04")
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

	decoder := newDecoder(r)
	var parsed taskNewActivityNewQ
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
	planID := r.PostForm.Get("PlanID")
	delete(r.PostForm, "PlanID")

	decoder := newDecoder(r)
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
	decoder := newDecoder(r)
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
	timeEnd, err := parseFormTime(r.FormValue("time_end"), r)
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
	timeEnd, err := parseFormTime(r.FormValue("time_end"), r)
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	timeStart, err := parseFormTime(r.FormValue("time_start"), r)
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
	loc := getTimeLocation(r)
	date, err := time.ParseInLocation("2006-01-02", r.PathValue("date"), loc)
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

	nColumns, columns := layout.Layout(events, 20*60) // minHeight is an arbitrary number

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

func mergeWindowToGraph[T any](w storage.Window[T], g *rdf2go.Graph, serializer func(T) (*rdf2go.Graph, rdf2go.Term)) error {
	const windowLength = 100
	for offset := 0; ; offset += windowLength {
		ts, err := w.Get(windowLength, offset)
		if err != nil {
			return err
		}
		for _, t := range ts {
			subG, _ := serializer(t)
			g.Merge(subG)
		}
		if len(ts) < 100 {
			break
		}
	}
	return nil
}

func (s *Server) getRDF(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if accept == "text/turtle" || accept == "application/ld+json" {
		w.Header().Set("Content-Type", accept)
	} else {
		accept = "text/turtle"
	}

	g := rdf2go.NewGraph(s.serializer.GraphURI())

	// === Task ===
	tw, err := s.st.TaskSearch("", time.Now(), r.Context())
	if err != nil {
		log.Printf("storage: storage: initial: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	err = mergeWindowToGraph(tw, g, s.serializer.TaskToRDF)
	if err != nil {
		log.Printf("storage: storage: merge: %s", err)
		http.Error(w, "storage error", 500)
		return
	}

	// === Activity ===
	aw, err := s.st.ActivityRange(time.Time{}, time.Now(), r.Context())
	if err != nil {
		log.Printf("storage: activity: initial: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	err = mergeWindowToGraph(aw, g, s.serializer.ActivityToRDF)
	if err != nil {
		log.Printf("storage: activity: merge: %s", err)
		http.Error(w, "storage error", 500)
		return
	}

	// === Plan ===
	pw, err := s.st.PlanRange(time.Time{}, time.Now(), r.Context())
	if err != nil {
		log.Printf("storage: plan: initial: %s", err)
		http.Error(w, "storage error", 500)
		return
	}
	err = mergeWindowToGraph(pw, g, s.serializer.PlanToRDF)
	if err != nil {
		log.Printf("storage: plan: merge: %s", err)
		http.Error(w, "storage error", 500)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("%s; charset=utf-8", accept))
	err = g.Serialize(w, accept)
	if err != nil {
		log.Printf("rdf serialization: %s", err)
		http.Error(w, "rdf serialization error", 500)
		return
	}
	return
}
