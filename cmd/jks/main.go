package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"nyiyui.ca/jks/data"
	"nyiyui.ca/jks/database"
)

func main() {
	a := app.New()
	w := a.NewWindow("jks")

	log.Printf("opening...")
	db, err := database.Open("db.sqlite3")
	if err != nil {
		panic(err)
	}
	log.Printf("migrating...")
	database.Migrate(db.DB)
	if err != nil {
		panic(err)
	}
	log.Printf("db ready.")

	tasksBinding, err := data.NewBinding[database.Task](db)
	if err != nil {
		panic(err)
	}
	log.Printf("binding ready.")

	tasksList := widget.NewListWithData(
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
	searchQuery := binding.NewString()
	selectedRowid := binding.NewInt()
	search := widget.NewEntry()
	search.Bind(searchQuery)
	tasksBinding.BindSearchQuery(searchQuery)
	tasksList.OnSelected = func(id widget.ListItemID) {
		row, err := tasksBinding.GetItem(int(id))
		if err != nil {
			log.Printf("failed to get row: %s", err)
			return
		}
		rowid := row.(*data.Row[database.Task]).Rowid
		log.Printf("select rowid=%d", rowid)
		err = selectedRowid.Set(rowid)
		if err != nil {
			log.Printf("failed to get row: %s", err)
			return
		}
	}
	//viewList:=
	w.SetContent(container.NewBorder(search, nil, nil, viewList, tasksList))
	w.Canvas().Focus(search)
	w.ShowAndRun()
}
