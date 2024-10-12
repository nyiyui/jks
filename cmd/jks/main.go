package main

import (
	"log"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/database"
	"nyiyui.ca/jks/ui"
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

	log.Printf("binding ready.")

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
	w.SetContent(tl)
	//w.Canvas().Focus(tl.Search)
	w.ShowAndRun()
}

func addTask(db *sqlx.DB) error {
	_, err := db.Exec(`INSERT INTO tasks (quick_title) VALUES (?)`, "new task")
	if err != nil {
		return err
	}
	return nil
}
