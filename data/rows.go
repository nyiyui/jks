package data

import (
	"database/sql"
	"errors"
	"log"
	"sync"

	"fyne.io/fyne/v2/data/binding"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/database"
)

type Rows2[T RowData] struct {
	db        *sqlx.DB
	listeners sync.Map
	rowCount  int

	getByRowid func(rowid int) *sqlx.Row
	getByIndex func(index int) *sqlx.Row
	getLength  func() *sql.Row
}

func NewRows2[T RowData](db *sqlx.DB) *Rows2[T] {
	r := &Rows2[T]{db: db}
	return r
}

func (r *Rows2[T]) AddListener(dl binding.DataListener) {
	dl.DataChanged()
	r.listeners.Store(dl, nil)
}

func (r *Rows2[T]) RemoveListener(dl binding.DataListener) {
	r.listeners.Delete(dl)
}

func (r *Rows2[T]) getData(rowid int) (T, error) {
	row := r.getByRowid(rowid)
	var data T
	err := row.StructScan(&data)
	if err != nil {
		var zero T
		return zero, err
	}
	return data, nil
}

func (r *Rows2[T]) GetItem(index int) (binding.DataItem, error) {
	row := r.getByIndex(index)
	var data T
	err := row.StructScan(&data)
	if err != nil {
		return nil, err
	}
	return &Row2[T]{
		r: r, Rowid: int(data.GetID()), data: data,
	}, nil
}

func (r *Rows2[T]) Length() int {
	return r.rowCount
}

func (r *Rows2[T]) UpdateLength() error {
	row := r.getLength()
	var count int
	err := row.Scan(&count)
	if err != nil {
		return err
	}
	r.rowCount = count
	log.Printf("length=%d", count)
	return nil
}

func (r *Rows2[T]) NotifyRowChange(rowid int64) {
	r.listeners.Range(func(key, value any) bool {
		if value == nil || value.(int64) == rowid {
			key.(binding.DataListener).DataChanged()
		}
		return true
	})
}

func (r *Rows2[T]) reloadAll() {
	r.listeners.Range(func(key, value any) bool {
		key.(binding.DataListener).DataChanged()
		return true
	})
}

type Row2[T RowData] struct {
	r     *Rows2[T]
	Rowid int
	data  T
	proxy *proxyDataListener
}

var _ GenericBinding[database.Task] = new(Row2[database.Task])

func (r *Row2[T]) callback() {
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

func (r *Row2[T]) AddListener(dl binding.DataListener) {
	dl.DataChanged()
	if r.proxy == nil {
		r.proxy = &proxyDataListener{r.callback, dl}
		r.r.listeners.Store(r.proxy, r.Rowid)
	} else {
		r.r.listeners.Store(dl, r.Rowid)
	}
}

func (r *Row2[T]) RemoveListener(dl binding.DataListener) {
	r.r.listeners.Delete(dl)
}

func (r *Row2[T]) Get() (T, error) {
	return r.data, nil
}

func (r *Row2[T]) Set(T) error {
	return errors.New("not implemented yet")
}
