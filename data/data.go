package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"

	"fyne.io/fyne/v2/data/binding"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
)

func NewBinding[T RowData](db *sqlx.DB) (*Rows[T], error) {
	c, err := db.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	r := &Rows[T]{
		db:        db,
		tableName: "tasks",
	}
	err = r.getLength()
	if err != nil {
		return nil, err
	}
	err = c.Raw(func(driverConn any) error {
		conn := driverConn.(*sqlite3.SQLiteConn)
		conn.RegisterUpdateHook(r.callback)
		return nil
	})
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
		log.Printf("searchCallback: getting new length failed", err)
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
		log.Print("select without query")
	} else {
		row = r.db.QueryRowx(`SELECT * FROM tasks WHERE quick_title like ? ORDER BY id ASC LIMIT 1 OFFSET ?`, r.searchQuery, index)
		log.Print("select with query")
	}
	err := row.StructScan(&data)
	if err != nil {
		return nil, err
	}
	return &Row[T]{
		r: r, rowid: int(data.GetID()), data: data,
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

func (r *Rows[T]) callback(op int, dbName, tableName string, rowid int64) {
	if tableName != r.tableName {
		return
	}
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
	rowid int
	data  T
	proxy *proxyDataListener
}

func (r *Row[T]) callback() {
	newData, err := r.r.getData(r.rowid)
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
		r.r.listeners.Store(r.proxy, r.rowid)
	} else {
		r.r.listeners.Store(dl, r.rowid)
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
