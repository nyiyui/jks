package main

import (
	"log"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/database"
	"nyiyui.ca/jks/ui"
)

func main() {
	a := app.New()
	w := a.NewWindow("Log Activity")

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

	tl, err := ui.NewTaskList(db)
	if err != nil {
		panic(err)
	}
	tl.SelectedRowid = binding.NewInt()
	//viewTask := widget.NewLabel("")
	//viewTask.Bind(selectedRowid)
	// addButton := widget.NewButton("Add Task", func() {
	// 	err := addTask(db)
	// 	if err != nil {
	// 		log.Printf("add task: %s", err)
	// 	}
	// })
	w.SetContent(container.NewBorder(widget.NewLabel("Log Activity"), nil, nil, nil, container.New(layout.NewGridLayout(2), widget.NewLabel("activity adding screen"), tl)))
	w.Canvas().Focus(tl.Search)
	w.ShowAndRun()
}

func addTask(db *sqlx.DB) error {
	_, err := db.Exec(`INSERT INTO tasks (quick_title) VALUES (?)`, "new task")
	if err != nil {
		return err
	}
	return nil
}
