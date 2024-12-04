package database

import (
	"context"
	"database/sql"
	"embed"
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

var _ storage.Storage = (*Database)(nil)

func (d *Database) ActivityAdd(a storage.Activity, ctx context.Context) (id int64, err error) {
	status := StatusInProgress
	if a.Done {
		status = StatusDone
	}
	res, err := d.DB.Exec(`INSERT INTO activity_log (task_id, location, time_start, time_end, status, note) VALUES (?, ?, ?, ?, ?, ?)`,
		a.TaskID,
		a.Location,
		a.TimeStart.Unix(),
		a.TimeEnd.Unix(),
		status,
		a.Note,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
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
	} else if orig.Status == StatusDone {
		status = StatusInProgress
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

func (d *Database) TaskGetPlans(id int64, limit, offset int, ctx context.Context) ([]storage.Plan, error) {
	ps := make([]Plan, limit)
	err := d.DB.Select(&ps, `SELECT * FROM plans WHERE task_id = ? LIMIT ? OFFSET ?`, id, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	ps2 := make([]storage.Plan, len(ps))
	for i := range ps {
		ps2[i] = planToStorage(ps[i])
	}
	return ps2, nil
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

func (d *Database) TaskSearch(query string, undoneAt time.Time, ctx context.Context) (storage.Window[storage.Task], error) {
	return &window3{d, undoneAt, query, ctx}, nil
}

type window3 struct {
	d     *Database
	now   time.Time
	query string
	ctx   context.Context
}

const deadlineWhere = `(deadline IS NULL OR UNIXEPOCH(deadline) >= ?)`
const queryWhere = `(quick_title like ? OR description like ?)`
const notDoneJoinWhere = `
JOIN activity_log ON (tasks.id = activity_log.task_id)
WHERE activity_log.time_end = (SELECT MAX(time_end) FROM activity_log WHERE activity_log.task_id = tasks.id)
AND activity_log.status != 3
`
const unionNoLogs = `
WHERE NOT EXISTS (SELECT * FROM activity_log WHERE activity_log.task_id = tasks.id)
`

func (w *window3) Get(limit, offset int) ([]storage.Task, error) {
	ts := make([]Task, limit)
	var err error
	if w.query == "" {
		err = w.d.DB.SelectContext(w.ctx, &ts,
			`SELECT * FROM (`+
				`SELECT tasks.* FROM tasks `+notDoneJoinWhere+` AND `+deadlineWhere+
				` UNION ALL `+
				`SELECT * FROM tasks `+unionNoLogs+` AND `+deadlineWhere+
				`) ORDER BY id ASC LIMIT ? OFFSET ?`,
			w.now.Unix(),
			w.now.Unix(),
			limit, offset)
	} else {
		err = w.d.DB.SelectContext(w.ctx, &ts,
			`SELECT * FROM (`+
				`SELECT tasks.* FROM tasks `+notDoneJoinWhere+` AND `+queryWhere+` AND `+deadlineWhere+
				` UNION ALL `+
				`SELECT * FROM tasks `+unionNoLogs+` AND `+queryWhere+` AND `+deadlineWhere+
				`) ORDER BY id ASC LIMIT ? OFFSET ?`,
			w.query, w.query, w.now.Unix(),
			w.query, w.query, w.now.Unix(),
			limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	ts2 := make([]storage.Task, len(ts))
	for i := range ts {
		ts2[i] = taskToStorage(ts[i])
	}
	return ts2, nil
}

func (w *window3) Close() error {
	return nil
}

func (d *Database) TaskAdd(v storage.Task, ctx context.Context) (id int64, err error) {
	res, err := d.DB.Exec(`INSERT INTO tasks (description, quick_title, deadline, due) VALUES (?, ?, ?, ?)`,
		v.Description,
		v.QuickTitle,
		v.Deadline,
		v.Due,
	)
	if err != nil {
		return
	}
	id, err = res.LastInsertId()
	return
}

func (d *Database) TaskEdit(v storage.Task, ctx context.Context) error {
	_, err := d.DB.Exec(`UPDATE tasks SET description = ?, quick_title = ?, deadline = ?, due = ? WHERE id = ?`,
		v.Description,
		v.QuickTitle,
		v.Deadline,
		v.Due,
		v.ID,
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

func (d *Database) PlanAdd(p storage.Plan, ctx context.Context) (id int64, err error) {
	res, err := d.DB.Exec(`INSERT INTO plans (task_id, activity_id, location, time_at_after, time_before, duration_ge, duration_lt) VALUES (?, ?, ?, ?, ?, ?, ?)`, p.TaskID, p.ActivityID, p.Location, p.TimeAtAfter.Unix(), p.TimeBefore.Unix(), p.DurationGe, p.DurationLt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *Database) PlanGet(id int64, ctx context.Context) (storage.Plan, error) {
	var v Plan
	err := d.DB.Get(&v, `SELECT * FROM plans WHERE id = ?`, id)
	if err != nil {
		return storage.Plan{}, fmt.Errorf("select: %w", err)
	}
	var activityID int64
	if v.ActivityID == nil {
		activityID = 0
	} else {
		activityID = *v.ActivityID
	}
	return storage.Plan{
		ID:          v.ID,
		TaskID:      v.TaskID,
		ActivityID:  activityID,
		Location:    v.Location,
		TimeAtAfter: v.TimeAtAfter,
		TimeBefore:  v.TimeBefore,
		DurationGe:  v.DurationGe,
		DurationLt:  v.DurationLt,
	}, nil
}

func (d *Database) PlanRange(a, b time.Time, ctx context.Context) (storage.Window[storage.Plan], error) {
	return &window4{d, a, b, ctx}, nil
}

type window4 struct {
	d    *Database
	a, b time.Time
	ctx  context.Context
}

func (w *window4) Get(limit, offset int) ([]storage.Plan, error) {
	ts := make([]Plan, limit)
	err := w.d.DB.SelectContext(w.ctx, &ts, `SELECT * FROM plans WHERE time_at_after >= ? AND time_before < ? LIMIT ? OFFSET ?`, w.a.Unix(), w.b.Unix(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	ts2 := make([]storage.Plan, len(ts))
	for i := range ts {
		ts2[i] = planToStorage(ts[i])
	}
	return ts2, nil
}

func (w *window4) Close() error {
	return nil
}

func planToStorage(p Plan) storage.Plan {
	var activityID int64
	if p.ActivityID == nil {
		activityID = 0
	} else {
		activityID = *p.ActivityID
	}
	return storage.Plan{
		ID:          p.ID,
		TaskID:      p.TaskID,
		ActivityID:  activityID,
		Location:    p.Location,
		TimeAtAfter: p.TimeAtAfter,
		TimeBefore:  p.TimeBefore,
		DurationGe:  p.DurationGe,
		DurationLt:  p.DurationLt,
	}
}

func (d *Database) PlanEdit(p storage.Plan, ctx context.Context) error {
	_, err := d.DB.ExecContext(ctx, `UPDATE plans SET task_id = ?, activity_id = ?, location = ?, time_at_after = ?, time_before = ?, duration_ge = ?, duration_lt = ? WHERE id = ?`, p.TaskID, p.ActivityID, p.Location, p.TimeAtAfter.Unix(), p.TimeBefore.Unix(), p.DurationGe, p.DurationLt, p.ID)
	return err
}

func (d *Database) Range(a, b time.Time, ctx context.Context) ([]storage.Task, []storage.Activity, []storage.Plan, error) {
	as := make([]Activity, 0)
	err := d.DB.SelectContext(ctx, &as, `SELECT * FROM activity_log WHERE time_start >= ? AND time_end < ?`, a.Unix(), b.Unix())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("select: %w", err)
	}
	activities := make([]storage.Activity, len(as))
	for i := range as {
		activities[i] = activityToStorage(as[i])
	}

	ps := make([]Plan, 0)
	err = d.DB.SelectContext(ctx, &ps, `SELECT * FROM plans WHERE time_at_after >= ? AND time_before < ?`, a.Unix(), b.Unix())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("select: %w", err)
	}
	plans := make([]storage.Plan, len(ps))
	for i := range ps {
		plans[i] = planToStorage(ps[i])
	}

	ts := make([]Task, 0)
	err = d.DB.SelectContext(ctx, &ts, `
SELECT tasks.* FROM tasks
JOIN plans ON (tasks.id = plans.task_id)
WHERE plans.time_at_after >= ? AND plans.time_before < ?
UNION ALL
SELECT tasks.* FROM tasks
JOIN activity_log ON (tasks.id = activity_log.task_id)
WHERE activity_log.time_start >= ? AND activity_log.time_end < ?
`, a.Unix(), b.Unix(), a.Unix(), b.Unix())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("select: %w", err)
	}
	tasks := make([]storage.Task, len(ts))
	for i := range ts {
		tasks[i] = taskToStorage(ts[i])
	}
	return tasks, activities, plans, nil
}
