package main

import (
	"log"

	"net/url"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/x/fyne/dialog"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/database"
	"nyiyui.ca/jks/ui"
)

func mustParse(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func main() {
	a := app.New()
	w := a.NewWindow("Log Activity")

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			dialog.ShowAbout("content", []*widget.Hyperlink{
				{Text: "Website", URL: mustParse("https://nyiyui.ca/jks")},
				{Text: "Source", URL: mustParse("https://github.com/nyiyui/jks")},
			}, a, w)
		}),
	)

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
	tl.SelectedTaskID = binding.NewInt()
	la, err := ui.NewLogActivity(db)
	if err != nil {
		panic(err)
	}
	la.BindTaskID(tl.SelectedTaskID)
	//viewTask := widget.NewLabel("")
	//viewTask.Bind(selectedRowid)
	// addButton := widget.NewButton("Add Task", func() {
	// 	err := addTask(db)
	// 	if err != nil {
	// 		log.Printf("add task: %s", err)
	// 	}
	// })
	w.SetContent(container.NewBorder(toolbar, nil, nil, nil, container.New(layout.NewGridLayout(2), la, tl)))
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
