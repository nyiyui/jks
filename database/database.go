package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"nyiyui.ca/jks/storage"
)

//go:embed migrations
var migrations embed.FS

type Database struct {
	DB *sqlx.DB
}

func Open(path string) (*sqlx.DB, error) {
	return sqlx.Open("sqlite3", path)
}

func Migrate(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return err
	}
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		panic(err) // shouldn't fail
	}
	m, err := migrate.NewWithInstance(
		"iofs", source,
		"sqlite3", driver)
	if err != nil {
		return err
	}
	err = m.Up()
	if err != nil {
		return err
	}
	return nil
}

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

var _ storage.Storage = (*Database)(nil)

func (d *Database) ActivityAdd(a storage.Activity, ctx context.Context) error {
	status := StatusInProgress
	if a.Done {
		status = StatusDone
	}
	_, err := d.DB.Exec(`INSERT INTO activity_log (task_id, location, time_start, time_end, status, note) VALUES (?, ?, ?, ?, ?, ?)`,
		a.TaskID,
		a.Location,
		a.TimeStart.Unix(),
		a.TimeEnd.Unix(),
		status,
		a.Note,
	)
	return err
}

func (d *Database) ActivityLatestN(ctx context.Context, n int) ([]storage.Activity, error) {
	var as []Activity
	err := d.DB.Select(&as, `SELECT * FROM activity_log ORDER BY time_end DESC LIMIT ? OFFSET 0`, n)
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	sas := make([]storage.Activity, 0, len(as))
	for _, a := range as {
		sas = append(sas, activityToStorage(a))
	}
	return sas, nil
}

func (d *Database) ActivityEdit(a storage.Activity, ctx context.Context) error {
	var orig Activity
	err := d.DB.Get(&orig, `SELECT * FROM activity_log WHERE id = ?`, a.ID)
	if err != nil {
		return fmt.Errorf("finding original: %w", err)
	}
	var status Status
	if a.Done {
		status = StatusDone
	} else {
		status = orig.Status
	}
	_, err = d.DB.Exec(`UPDATE activity_log SET task_id = ?, location = ?, time_start = ?, time_end = ?, status = ?, note = ? WHERE id = ?`,
		a.TaskID,
		a.Location,
		a.TimeStart.Unix(),
		a.TimeEnd.Unix(),
		status,
		a.Note,
		a.ID,
	)
	return err
}

func (d *Database) ActivityGet(id int64, ctx context.Context) (storage.Activity, error) {
	var v Activity
	err := d.DB.Get(&v, `SELECT * FROM activity_log WHERE id = ?`, id)
	if err != nil {
		return storage.Activity{}, fmt.Errorf("select: %w", err)
	}
	return activityToStorage(v), nil
}

func (d *Database) TaskGet(id int64, ctx context.Context) (storage.Task, error) {
	var t Task
	err := d.DB.Get(&t, `SELECT * FROM tasks WHERE id = ?`, id)
	if err != nil {
		return storage.Task{}, fmt.Errorf("select: %w", err)
	}
	return storage.Task{
		ID:          t.ID,
		Description: t.Description,
		QuickTitle:  t.QuickTitle,
		Deadline:    t.Deadline,
		Due:         t.Due,
	}, nil
}

func (d *Database) TaskGetActivities(id int64, ctx context.Context) (storage.Window[storage.Activity], error) {
	return &window{d, id}, nil
}

type window struct {
	d  *Database
	id int64
}

func (w *window) Get(limit, offset int) ([]storage.Activity, error) {
	ts := make([]Activity, limit)
	err := w.d.DB.Select(&ts, `SELECT * FROM activity_log WHERE task_id = ? LIMIT ? OFFSET ?`, w.id, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	ts2 := make([]storage.Activity, len(ts))
	for i := range ts {
		ts2[i] = activityToStorage(ts[i])
	}
	return ts2, nil
}

func (w *window) Close() error {
	return nil
}

func taskToStorage(t Task) storage.Task {
	return storage.Task{
		ID:          t.ID,
		Description: t.Description,
		QuickTitle:  t.QuickTitle,
		Deadline:    t.Deadline,
		Due:         t.Due,
	}
}

func activityToStorage(a Activity) storage.Activity {
	done := false
	if a.Status == StatusDone {
		done = true
	}
	return storage.Activity{
		ID:        a.ID,
		TaskID:    a.TaskID,
		Location:  a.Location,
		TimeStart: a.TimeStart,
		TimeEnd:   a.TimeEnd,
		Done:      done,
		Note:      a.Note,
	}
}

func (d *Database) TaskSearch(query string, ctx context.Context) (storage.Window[storage.Task], error) {
	return nil, errors.New("not implemented")
}

func (d *Database) TaskAdd(v storage.Task, ctx context.Context) error {
	_, err := d.DB.Exec(`INSERT INTO tasks (description, quick_title, deadline, due) VALUES (?, ?, ?, ?)`,
		v.Description,
		v.QuickTitle,
		v.Deadline,
		v.Due,
	)
	return err
}

func (d *Database) ActivityRange(a, b time.Time, ctx context.Context) (storage.Window[storage.Activity], error) {
	return &window2{d, a, b, ctx}, nil
}

type window2 struct {
	d         *Database
	timeStart time.Time
	timeEnd   time.Time
	ctx       context.Context
}

func (w *window2) Get(limit, offset int) ([]storage.Activity, error) {
	ts := make([]Activity, limit)
	err := w.d.DB.SelectContext(w.ctx, &ts, `SELECT * FROM activity_log WHERE time_start >= ? AND time_end < ? LIMIT ? OFFSET ?`, w.timeStart.Unix(), w.timeEnd.Unix(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	ts2 := make([]storage.Activity, len(ts))
	for i := range ts {
		ts2[i] = activityToStorage(ts[i])
	}
	return ts2, nil
}

func (w *window2) Close() error {
	return nil
}
