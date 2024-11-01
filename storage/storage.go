package storage

import (
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
	ActivityAdd(a Activity) error
}
