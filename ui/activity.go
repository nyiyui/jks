package ui

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/data"
	"nyiyui.ca/jks/database"
)

type LogActivity struct {
	widget.BaseWidget
	taskIDData       binding.Int
	db               *sqlx.DB
	taskID           int64
	task             database.Task
	lastActivityData data.Activity
	lastActivity     *Activity
}

func NewLogActivity(db *sqlx.DB) (*LogActivity, error) {
	la := new(LogActivity)
	la.ExtendBaseWidget(la)
	la.db = db
	la.lastActivityData = data.NewActivity(db, 0)
	la.lastActivity = NewActivity()
	la.lastActivity.Bind(la.lastActivityData)
	la.lastActivity.Disable()
	return la, nil
}

func (la *LogActivity) BindTaskID(taskID binding.Int) {
	la.taskIDData = taskID
	la.taskIDData.AddListener(la)
}

func (la *LogActivity) UnbindTaskID(taskID binding.Int) {
	la.taskIDData.RemoveListener(la)
}

func (la *LogActivity) refresh() {
	log.Printf("task ID = %d", la.taskID)
	row := la.db.QueryRowx(`SELECT * FROM tasks WHERE id = ?`, la.taskID)
	var task database.Task
	err := row.StructScan(&task)
	if errors.Is(err, sql.ErrNoRows) {
		log.Print("no task id")
		la.lastActivity.Disable()
		return
	} else if err != nil {
		log.Printf("failed to get task: %s", err)
		return
	}
	la.task = task

	row = la.db.QueryRowx(`SELECT (id) FROM activity_log WHERE task_id = ? ORDER BY time_end DESC LIMIT 1 OFFSET 0`, la.taskID)
	var rowid int64
	err = row.Scan(&rowid)
	if errors.Is(err, sql.ErrNoRows) {
		log.Print("no last activity")
		la.lastActivity.Disable()
		return
	} else if err != nil {
		log.Printf("failed to get last activity: %s", err)
		return
	}
	la.lastActivity.Enable()
	err = la.lastActivityData.SetRowid(rowid)
	if err != nil {
		log.Printf("failed to set last activity: %s", err)
		return
	}
}

func (la *LogActivity) DataChanged() {
	taskID, err := la.taskIDData.Get()
	if err != nil {
		log.Printf("error getting task ID: %s", err)
	}
	la.taskID = int64(taskID)
	la.refresh()
}

func (la *LogActivity) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewAppTabs(
		container.NewTabItem("Extend Last Activity", la.lastActivity),
	)
	return widget.NewSimpleRenderer(c)
}

type Activity struct {
	widget.BaseWidget
	widget.DisableableWidget
	activity      database.Activity
	idLabel       *widget.Label
	idValue       *widget.Label
	locationLabel *widget.Label
	locationValue *widget.Entry
	binding       data.Activity
}

func NewActivity() *Activity {
	a := new(Activity)
	a.idLabel = widget.NewLabel("ID")
	a.idValue = widget.NewLabel("")
	a.locationLabel = widget.NewLabel("Location")
	a.locationValue = widget.NewEntry()
	a.locationValue.OnChanged = func(newLocation string) {
		a.activity.Location = newLocation
		err := a.binding.Set(a.activity)
		if err != nil {
			log.Printf("failed to set activity binding: %s", err)
		}
		return
	}
	return a
}

func (a *Activity) Bind(data data.Activity) {
	a.binding = data
	a.binding.AddListener(a)
}

func (a *Activity) DataChanged() {
	log.Printf("Activity DataChanged")
	activity, err := a.binding.Get()
	if err != nil {
		log.Printf("failed to get ID: %s", err)
		return
	}
	if activity == (database.Activity{}) {
		return
	}
	a.activity = activity
	a.idValue.Text = fmt.Sprint(a.activity.ID)
	a.idValue.Refresh()
	a.locationValue.Text = a.activity.Location
	a.locationValue.Refresh()
}

func (a *Activity) Disable() {
	a.DisableableWidget.Disable()
	a.refresh()
}

func (a *Activity) Enable() {
	a.DisableableWidget.Enable()
	a.refresh()
}

func (a *Activity) refresh() {
	if a.Disabled() {
		a.locationValue.Disable()
	} else {
		a.locationValue.Enable()
	}
}

func (a *Activity) CreateRenderer() fyne.WidgetRenderer {
	form := container.New(layout.NewFormLayout(), a.idLabel, a.idValue, a.locationLabel, a.locationValue)
	c := container.New(layout.NewStackLayout(), form)
	return widget.NewSimpleRenderer(c)
}
