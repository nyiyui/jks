package ui

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/data"
	"nyiyui.ca/jks/database"
)

type TaskRow2 struct {
	r *data.Row2[database.Task]
}

func (tr *TaskRow2) AddListener(dl binding.DataListener) {
	tr.r.AddListener(dl)
}

func (tr *TaskRow2) RemoveListener(dl binding.DataListener) {
	tr.r.RemoveListener(dl)
}

func (tr *TaskRow2) Get() (string, error) {
	r, err := tr.r.Get()
	if err != nil {
		return "", err
	}
	return r.QuickTitle, nil
}

func (tr *TaskRow2) Set(string) error {
	return errors.New("not implemented")
}

type TaskList struct {
	widget.BaseWidget
	List           *widget.List
	Search         *widget.Entry
	SearchQuery    binding.String
	SelectedTaskID binding.Int
	container      *fyne.Container

	// uiTime is the time at which the UI is conceptually created. Tasks with a deadline before this time should not be displayed.
	uiTime time.Time
}

func NewTaskList(db *sqlx.DB) (*TaskList, error) {
	tl := new(TaskList)
	tl.uiTime = time.Now()
	tasksBinding := data.NewRows2[database.Task](db, func(rowid int) *sqlx.Row {
		if query, _ := tl.SearchQuery.Get(); query != "" {
			query2 := "%" + query + "%"
			return db.QueryRowx(`SELECT * FROM tasks WHERE (quick_title like ? OR description like ?) AND (deadline IS NULL OR UNIXEPOCH(deadline) >= ?) AND id = ?`, query2, query2, tl.uiTime.Unix(), rowid)
		} else {
			return db.QueryRowx(`SELECT * FROM tasks WHERE (deadline IS NULL OR UNIXEPOCH(deadline) >= ?) AND id = ?`, tl.uiTime.Unix(), rowid)
		}
	}, func(index int) *sqlx.Row {
		if query, _ := tl.SearchQuery.Get(); query != "" {
			query2 := "%" + query + "%"
			return db.QueryRowx(`SELECT * FROM tasks WHERE (quick_title like ? OR description like ?) AND (deadline IS NULL OR UNIXEPOCH(deadline) >= ?) ORDER BY id ASC LIMIT 1 OFFSET ?`, query2, query2, tl.uiTime.Unix(), index)
		} else {
			return db.QueryRowx(`SELECT * FROM tasks WHERE (deadline IS NULL OR UNIXEPOCH(deadline) >= ?) ORDER BY id ASC LIMIT 1 OFFSET ?`, tl.uiTime.Unix(), index)
		}
	}, func() *sql.Row {
		if query, _ := tl.SearchQuery.Get(); query != "" {
			query2 := "%" + query + "%"
			return db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE (quick_title like ? OR description like ?) AND (deadline IS NULL OR UNIXEPOCH(deadline) >= ?)`, query2, query2, tl.uiTime.Unix())
		} else {
			return db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE (deadline IS NULL OR UNIXEPOCH(deadline) >= ?)`, tl.uiTime.Unix())
		}
	})
	tl.List = widget.NewListWithData(
		tasksBinding,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			row := item.(*data.Row2[database.Task])
			label := obj.(*widget.Label)
			label.Bind(&TaskRow2{row})
		},
	)

	// search entry setup
	tl.SearchQuery = binding.NewString()
	tl.Search = widget.NewEntry()
	tl.Search.Bind(tl.SearchQuery)
	tl.SearchQuery.AddListener(tasksBinding)

	tl.List.OnSelected = func(id widget.ListItemID) {
		if tl.SelectedTaskID == nil {
			return
		}
		row, err := tasksBinding.GetItem(int(id))
		if err != nil {
			log.Printf("failed to get row: %s", err)
			return
		}
		rowid := row.(*data.Row2[database.Task]).Rowid
		log.Printf("select rowid=%d", rowid)
		err = tl.SelectedTaskID.Set(rowid)
		if err != nil {
			log.Printf("failed to get row: %s", err)
			return
		}
	}
	tl.ExtendBaseWidget(tl)

	tl.container = container.NewBorder(tl.Search, nil, nil, nil, tl.List)

	return tl, nil
}

func (tl *TaskList) MinSize() fyne.Size {
	return tl.container.MinSize()
}

func (tl *TaskList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(tl.container)
}
