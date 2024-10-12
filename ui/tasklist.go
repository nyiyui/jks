package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/data"
	"nyiyui.ca/jks/database"
)

type TaskList struct {
	widget.BaseWidget
	List           *widget.List
	Search         *widget.Entry
	SearchQuery    binding.String
	SelectedTaskID binding.Int
}

func NewTaskList(db *sqlx.DB) (*TaskList, error) {
	tasksBinding, err := data.NewBinding[database.Task](db)
	if err != nil {
		return nil, err
	}

	tl := new(TaskList)
	tl.List = widget.NewListWithData(
		tasksBinding,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			row := item.(*data.Row[database.Task])
			label := obj.(*widget.Label)
			label.Bind(row)
		},
	)

	// search entry setup
	tl.SearchQuery = binding.NewString()
	tl.Search = widget.NewEntry()
	tl.Search.Bind(tl.SearchQuery)
	tasksBinding.BindSearchQuery(tl.SearchQuery)

	tl.List.OnSelected = func(id widget.ListItemID) {
		if tl.SelectedTaskID == nil {
			return
		}
		row, err := tasksBinding.GetItem(int(id))
		if err != nil {
			log.Printf("failed to get row: %s", err)
			return
		}
		rowid := row.(*data.Row[database.Task]).Rowid
		log.Printf("select rowid=%d", rowid)
		err = tl.SelectedTaskID.Set(rowid)
		if err != nil {
			log.Printf("failed to get row: %s", err)
			return
		}
	}
	tl.ExtendBaseWidget(tl)

	return tl, nil
}

func (tl *TaskList) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(tl.Search, nil, nil, nil, tl.List)
	return widget.NewSimpleRenderer(c)
}
