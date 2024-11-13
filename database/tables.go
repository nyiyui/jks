package database

import "time"

type Task struct {
	ID          int64
	Description string
	QuickTitle  string `db:"quick_title"`

	// Deadline is the time after which this task is useless to complete.
	// For example, studying for an exam after the exam itself is useless (for the purpose of scoring well on the exam).
	// In this case, the deadline would be the exam start time.
	// In the future, this may become a reference to another task, such that once that task is started, this task is useless to complete..
	Deadline *time.Time `db:"deadline"`
	Due      *time.Time `db:"due"`
}

func (t Task) GetID() int64 { return t.ID }

func (t Task) String() string {
	return t.QuickTitle
}

type Activity struct {
	ID        int64
	TaskID    int64 `db:"task_id"`
	Location  string
	TimeStart time.Time `db:"time_start"`
	TimeEnd   time.Time `db:"time_end"`
	Status    Status
	Note      string
}

func (a Activity) GetID() int64 { return a.ID }

type Status int

const (
	StatusUnknown Status = iota
	StatusNotStarted
	StatusInProgress
	StatusDone
)

var StatusNames = [...]string{
	StatusUnknown:    "Unknown",
	StatusNotStarted: "Not Started",
	StatusInProgress: "In Progress",
	StatusDone:       "Done",
}

func (s Status) String() string {
	return StatusNames[s]
}

type Plan struct {
	ID          int64
	TaskID      int64  `db:"task_id"`
	ActivityID  *int64 `db:"activity_id"`
	Location    string
	TimeAtAfter time.Time     `db:"time_at_after"`
	TimeBefore  time.Time     `db:"time_before"`
	DurationGe  time.Duration `db:"duration_ge"`
	DurationLt  time.Duration `db:"duration_lt"`
}

func (p Plan) GetID() int64 { return p.ID }
