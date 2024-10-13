package data

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/database"
)

func NewBinding[T RowData](db *sqlx.DB) (*Rows[T], error) {
	r := &Rows[T]{
		db:        db,
		tableName: "tasks",
	}
	err := r.getLength()
	if err != nil {
		return nil, err
	}
	return r, nil
}

type RowData interface {
	fmt.Stringer
	GetID() int64
}

type Rows[T RowData] struct {
	db            *sqlx.DB
	tableName     string
	listeners     sync.Map
	rowCount      int
	searchQuery   string
	searchBinding binding.String
}

func (r *Rows[T]) AddListener(dl binding.DataListener) {
	dl.DataChanged()
	r.listeners.Store(dl, nil)
}

func (r *Rows[T]) RemoveListener(dl binding.DataListener) {
	r.listeners.Delete(dl)
}

func (r *Rows[T]) BindSearchQuery(data binding.String) {
	data.AddListener(&callbackListener{r.searchCallback})
	r.searchBinding = data
}

func (r *Rows[T]) searchCallback() {
	query, err := r.searchBinding.Get()
	if err != nil {
		log.Printf("searchCallback: getting string failed: %s", err)
		return
	}
	r.searchQuery = query
	err = r.getLength()
	if err != nil {
		log.Printf("searchCallback: getting new length failed: %s", err)
	}
	r.reloadAll()
}

func (r *Rows[T]) getData(rowid int) (T, error) {
	var row *sqlx.Row
	if r.searchQuery == "" {
		row = r.db.QueryRowx(`SELECT * FROM tasks WHERE id = ?`, rowid)
	} else {
		row = r.db.QueryRowx(`SELECT * FROM tasks WHERE quick_title like ? AND id = ?`, r.searchQuery, rowid)
	}
	var data T
	err := row.StructScan(&data)
	if err != nil {
		var zero T
		return zero, err
	}
	return data, nil
}

func (r *Rows[T]) GetItem(index int) (binding.DataItem, error) {
	var data T
	var row *sqlx.Row
	if r.searchQuery == "" {
		row = r.db.QueryRowx(`SELECT * FROM tasks ORDER BY id ASC LIMIT 1 OFFSET ?`, index)
	} else {
		row = r.db.QueryRowx(`SELECT * FROM tasks WHERE quick_title like ? ORDER BY id ASC LIMIT 1 OFFSET ?`, r.searchQuery, index)
	}
	err := row.StructScan(&data)
	if err != nil {
		return nil, err
	}
	return &Row[T]{
		r: r, Rowid: int(data.GetID()), data: data,
	}, nil
}

func (r *Rows[T]) Length() int {
	return r.rowCount
}

func (r *Rows[T]) getLength() error {
	var count int
	var rows *sql.Row
	if r.searchQuery == "" {
		rows = r.db.QueryRow(`SELECT COUNT(*) FROM tasks`)
	} else {
		rows = r.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE quick_title like ?`, r.searchQuery)
	}
	err := rows.Scan(&count)
	if err != nil {
		return err
	}
	r.rowCount = count
	log.Printf("length=%d", count)
	return nil
}

func (r *Rows[T]) NotifyRowChange(rowid int64) {
	r.listeners.Range(func(key, value any) bool {
		if value == nil || value.(int64) == rowid {
			key.(binding.DataListener).DataChanged()
		}
		return true
	})
}

func (r *Rows[T]) reloadAll() {
	r.listeners.Range(func(key, value any) bool {
		key.(binding.DataListener).DataChanged()
		return true
	})
}

type proxyDataListener struct {
	callback func()
	dl       binding.DataListener
}

func (p *proxyDataListener) DataChanged() {
	p.callback()
	p.dl.DataChanged()
}

type Row[T RowData] struct {
	r     *Rows[T]
	Rowid int
	data  T
	proxy *proxyDataListener
}

func (r *Row[T]) callback() {
	newData, err := r.r.getData(r.Rowid)
	if err != nil {
		// assume this is a synchronization issue
		// log.Printf("failed to get data: %s", err)
		var zero T
		r.data = zero
		return
	}
	r.data = newData
}

func (r *Row[T]) AddListener(dl binding.DataListener) {
	dl.DataChanged()
	if r.proxy == nil {
		r.proxy = &proxyDataListener{r.callback, dl}
		r.r.listeners.Store(r.proxy, r.Rowid)
	} else {
		r.r.listeners.Store(dl, r.Rowid)
	}
}

func (r *Row[T]) RemoveListener(dl binding.DataListener) {
	r.r.listeners.Delete(dl)
}

func (r *Row[T]) Get() (string, error) {
	return fmt.Sprint(r.data), nil
}

func (r *Row[T]) Set(string) error {
	return errors.New("not implemented yet")
}

type Activity interface {
	binding.DataItem
	Get() (database.Activity, error)
	Set(database.Activity) error
	SetRowid(int64) error
	SetLocation(string) error
	SetTimeStart(time.Time) error
	SetTimeEnd(time.Time) error
}

type baseBinding struct {
	listeners sync.Map
}

func (b *baseBinding) notifyAllListeners() {
	b.listeners.Range(func(key, value any) bool {
		key.(binding.DataListener).DataChanged()
		return true
	})
}

func (b *baseBinding) AddListener(dl binding.DataListener) {
	b.listeners.Store(dl, nil)
}

func (b *baseBinding) RemoveListener(dl binding.DataListener) {
	b.listeners.Delete(dl)
}

type activityBinding struct {
	*baseBinding
	db    *sqlx.DB
	rowid int64
}

// NewActivity returns a new Activity with the appropriate rowid.
// Set rowid to zero if rowid is not set.
func NewActivity(db *sqlx.DB, rowid int64) Activity {
	return &activityBinding{new(baseBinding), db, rowid}
}

func (a *activityBinding) Get() (database.Activity, error) {
	if a.rowid == 0 {
		return database.Activity{}, nil
	}
	row := a.db.QueryRowx(`SELECT * FROM activity_log WHERE id = ? LIMIT 1 OFFSET 0`, a.rowid)
	var data database.Activity
	err := row.StructScan(&data)
	if err != nil {
		return database.Activity{}, err
	}
	return data, nil
}

func (a *activityBinding) SetRowid(rowid int64) error {
	a.rowid = rowid
	a.notifyAllListeners()
	return nil
}

func (a *activityBinding) Set(data database.Activity) error {
	_, err := a.db.Exec(
		`UPDATE activity_log SET task_id = ?, location = ?, time_start = ?, time_end = ? WHERE id = ?`,
		data.TaskID,
		data.Location,
		data.TimeStart.Unix(),
		data.TimeEnd.Unix(),
		a.rowid,
	)
	if err != nil {
		return err
	}
	a.notifyAllListeners()
	return nil
}

func (a *activityBinding) SetTimeStart(timeStart time.Time) error {
	_, err := a.db.Exec(
		`UPDATE activity_log SET time_start = ? WHERE id = ?`,
		timeStart.Unix(),
		a.rowid,
	)
	if err != nil {
		return err
	}
	a.notifyAllListeners()
	return nil
}

func (a *activityBinding) SetTimeEnd(timeEnd time.Time) error {
	_, err := a.db.Exec(
		`UPDATE activity_log SET time_end = ? WHERE id = ?`,
		timeEnd.Unix(),
		a.rowid,
	)
	if err != nil {
		return err
	}
	a.notifyAllListeners()
	return nil
}

func (a *activityBinding) SetLocation(location string) error {
	_, err := a.db.Exec(
		`UPDATE activity_log SET location = ? WHERE id = ?`,
		location,
		a.rowid,
	)
	if err != nil {
		return err
	}
	a.notifyAllListeners()
	return nil
}
