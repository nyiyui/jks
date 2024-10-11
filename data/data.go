package data

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"fyne.io/fyne/v2/data/binding"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
)

func NewBinding[T RowData](db *sqlx.DB) (binding.DataList, error) {
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
	db        *sqlx.DB
	tableName string
	listeners sync.Map
	rowCount  int
}

func (r *Rows[T]) AddListener(dl binding.DataListener) {
	dl.DataChanged()
	r.listeners.Store(dl, nil)
}

func (r *Rows[T]) RemoveListener(dl binding.DataListener) {
	r.listeners.Delete(dl)
}

func (b *Rows[T]) getData(rowid int) (T, error) {
	row := b.db.QueryRowx(`SELECT * FROM tasks WHERE id = ?`, rowid)
	var data T
	err := row.Scan(data)
	if err != nil {
		var zero T
		return zero, err
	}
	return data, nil
}

func (b *Rows[T]) GetItem(index int) (binding.DataItem, error) {
	tasks := make([]T, 1)
	err := b.db.Select(&tasks, `SELECT * FROM tasks ORDER BY id ASC LIMIT 1 OFFSET ?`, index)
	if err != nil {
		return nil, err
	}
	log.Printf("GetItem(%d) with rowid %d", index, tasks[0].GetID())
	return &Row[T]{
		r: b, rowid: int(tasks[0].GetID()), data: tasks[0],
	}, nil
}

func (r *Rows[T]) Length() int {
	return r.rowCount
}

func (r *Rows[T]) getLength() error {
	var count int
	rows := r.db.QueryRow(`SELECT COUNT(*) FROM tasks`)
	err := rows.Scan(&count)
	if err != nil {
		return err
	}
	r.rowCount = count
	log.Printf("length=%d", count)
	return nil
}

func (b *Rows[T]) callback(op int, dbName, tableName string, rowid int64) {
	if tableName != b.tableName {
		return
	}
	b.listeners.Range(func(key, value any) bool {
		if value == nil || value.(int64) == rowid {
			key.(binding.DataListener).DataChanged()
		}
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
		log.Printf("failed to get data: %s", err)
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
