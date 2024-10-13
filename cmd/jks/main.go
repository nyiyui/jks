package main

import (
	"log"

	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/x/fyne/dialog"
	"nyiyui.ca/jks/data"
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
			dialog.ShowAbout("jks is a function that helps convert tasks and requirements into a concrete schedule.", []*widget.Hyperlink{
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
	taskBinding := data.NewTask(db, 0)
	taskInfo := ui.NewTaskInfo(taskBinding)
	tl.SelectedTaskID = binding.NewInt()
	newProxy(tl.SelectedTaskID, taskBinding)
	la, err := ui.NewLogActivity(db, w)
	if err != nil {
		panic(err)
	}
	la.BindTaskID(tl.SelectedTaskID)
	w.SetContent(container.NewBorder(toolbar, nil, nil, nil,
		container.New(layout.NewGridLayout(2),
			la,
			container.NewVBox(tl, taskInfo),
		),
	))
	w.Canvas().Focus(tl.Search)
	w.ShowAndRun()
}

type proxy struct {
	source  data.GenericBinding[int]
	binding data.Task
}

func newProxy(source data.GenericBinding[int], binding data.Task) *proxy {
	p := new(proxy)
	p.source = source
	p.binding = binding
	p.source.AddListener(p)
	return p
}

func (p *proxy) DataChanged() {
	taskID, err := p.source.Get()
	if err != nil {
		fyne.LogError("get task id: %s", err)
		return
	}
	p.binding.SetRowid(int64(taskID))
	log.Printf("set rowid %d", taskID)
}
