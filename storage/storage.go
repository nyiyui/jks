package storage

import (
	"context"
	"net/url"
	"time"

	"nyiyui.ca/jks/linkdata"
)

type Task struct {
	ID          int64
	Description string
	QuickTitle  string

	// Deadline is the time after which this task is useless to complete.
	// For example, studying for an exam after the exam itself is useless (for the purpose of scoring well on the exam).
	// In this case, the deadline would be the exam start time.
	// In the future, this may become a reference to another task, such that once that task is started, this task is useless to complete..
	Deadline *time.Time
	Due      *time.Time
}

type Activity struct {
	ID        int64
	TaskID    int64
	Location  string
	TimeStart time.Time
	TimeEnd   time.Time
	Done      bool
	Note      string
}

func (a Activity) Layout() (top int, height int) {
	start := a.TimeStart.Unix()
	end := a.TimeEnd.Unix()
	return int(start), int(end - start)
}

type Plan struct {
	ID     int64
	TaskID int64
	// ActivityID is zero if there is no associated activity.
	ActivityID  int64
	Location    string
	TimeAtAfter time.Time
	TimeBefore  time.Time
	DurationGe  time.Duration
	DurationLt  time.Duration
}

func (p Plan) Layout() (top int, height int) {
	start := p.TimeAtAfter.Unix()
	end := p.TimeBefore.Unix()
	return int(start), int(end - start)
}

type Storage interface {
	ActivityAdd(a Activity, ctx context.Context) (id int64, err error)
	ActivityLatestN(ctx context.Context, n int) ([]Activity, error)
	ActivityGet(id int64, ctx context.Context) (Activity, error)
	ActivityRange(a, b time.Time, ctx context.Context) (Window[Activity], error)
	ActivityEdit(a Activity, ctx context.Context) error

	PlanAdd(p Plan, ctx context.Context) (id int64, err error)
	PlanGet(id int64, ctx context.Context) (Plan, error)
	PlanRange(a, b time.Time, ctx context.Context) (Window[Plan], error)
	PlanEdit(p Plan, ctx context.Context) error

	TaskGet(id int64, ctx context.Context) (Task, error)
	TaskGetActivities(id int64, ctx context.Context) ([]Activity, error)
	TaskGetPlans(id int64, limit, offset int, ctx context.Context) ([]Plan, error)
	TaskSearch(query string, undoneAt time.Time, ctx context.Context) (Window[Task], error)
	TaskAdd(t Task, ctx context.Context) (id int64, err error)
	TaskEdit(t Task, ctx context.Context) error

	// Range returns activities and plans returned by PlanRange and ActivityRange.
	// Tasks are the tasks referred to by each activity and plan.
	Range(a, b time.Time, ctx context.Context) ([]Task, []Activity, []Plan, error)

	ReplaceLinks(source *url.URL, links []linkdata.Link, ctx context.Context) error
	GetLinks(source *url.URL, ctx context.Context) ([]linkdata.Link, error)
	GetBacklinks(destination *url.URL, ctx context.Context) ([]linkdata.Backlink, error)
}

type Window[T any] interface {
	Get(limit, offset int) ([]T, error)
	// Close is idempotent.
	Close() error
}
