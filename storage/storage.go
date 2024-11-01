package storage

import (
	"context"
	"time"
)

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
	ActivityLatest(ctx context.Context) (Activity, error)
	ActivityEdit(a Activity, ctx context.Context) error
}
