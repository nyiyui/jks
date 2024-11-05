package storage

import (
	"context"
	"time"
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

type Storage interface {
	ActivityAdd(a Activity, ctx context.Context) error
	ActivityLatestN(ctx context.Context, n int) ([]Activity, error)
	ActivityGet(id int64, ctx context.Context) (Activity, error)
	ActivityRange(a, b time.Time, ctx context.Context) (Window[Activity], error)
	ActivityEdit(a Activity, ctx context.Context) error
	TaskGet(id int64, ctx context.Context) (Task, error)
	TaskGetActivities(id int64, ctx context.Context) (Window[Activity], error)
	TaskSearch(query string, ctx context.Context) (Window[Task], error)
	TaskAdd(t Task, ctx context.Context) error
}

type Window[T any] interface {
	Get(limit, offset int) ([]T, error)
	// Close is idempotent.
	Close() error
}
