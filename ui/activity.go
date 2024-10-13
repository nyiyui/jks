package ui

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/data"
	"nyiyui.ca/jks/database"
	"nyiyui.ca/jks/xwidget"
)

type LogActivity struct {
	widget.BaseWidget
	taskIDData       binding.Int
	db               *sqlx.DB
	taskID           int64
	task             database.Task
	lastActivityData data.Activity
	lastActivity     *Activity

	selectTaskHint *widget.Label
}

func NewLogActivity(db *sqlx.DB) (*LogActivity, error) {
	la := new(LogActivity)
	la.ExtendBaseWidget(la)
	la.db = db
	la.lastActivityData = data.NewActivity(db, 0)
	la.lastActivity = NewActivity(la.lastActivityData)
	la.lastActivity.Disable()
	la.selectTaskHint = widget.NewLabel("")
	la.selectTaskHint.Importance = widget.LowImportance
	la.refresh()
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
		la.selectTaskHint.Text = "Select a task to extend."
		la.selectTaskHint.Refresh()
		return
	} else if err != nil {
		log.Printf("failed to get task: %s", err)
		return
	}
	la.task = task
	la.selectTaskHint.Text = ""
	la.selectTaskHint.Refresh()

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
		container.NewTabItem("Extend Last", container.NewVBox(la.lastActivity, la.selectTaskHint)),
		//container.NewTabItem("Add New", la.newActivity),
	)
	return widget.NewSimpleRenderer(c)
}

type Activity struct {
	widget.BaseWidget
	widget.DisableableWidget
	activity database.Activity

	idLabel *widget.Label
	idValue *widget.Label

	locationLabel   *widget.Label
	locationValue   *widget.Entry
	locationBinding data.GenericBinding[string]

	timeStartLabel   *widget.Label
	timeStartValue   *xwidget.DateTime
	timeStartBinding data.GenericBinding[time.Time]

	timeEndLabel   *widget.Label
	timeEndValue   *xwidget.DateTime
	timeEndBinding data.GenericBinding[time.Time]

	binding data.Activity

	container *fyne.Container
}

func NewActivity(activityBinding data.Activity) *Activity {
	a := new(Activity)
	a.binding = activityBinding
	a.binding.AddListener(a)
	a.idLabel = widget.NewLabel("ID")
	a.idValue = widget.NewLabel("")
	a.locationBinding = data.NewSubBindingImperative[data.Activity, database.Activity, string](
		a.binding,
		func(activity database.Activity) (string, error) {
			return activity.Location, nil
		},
		func(binding data.Activity, location string) error {
			return binding.SetLocation(location)
		},
	)
	a.locationLabel = widget.NewLabel("Location")
	a.locationValue = widget.NewEntry()
	a.locationValue.Bind(a.locationBinding)
	// a.locationValue.OnChanged = func(newLocation string) {
	// 	a.activity.Location = newLocation
	// 	err := a.binding.Set(a.activity)
	// 	if err != nil {
	// 		log.Printf("failed to set activity binding: %s", err)
	// 	}
	// 	return
	// }
	a.timeStartBinding = data.NewSubBindingImperative[data.Activity, database.Activity, time.Time](
		a.binding,
		func(activity database.Activity) (time.Time, error) {
			return activity.TimeStart, nil
		},
		func(binding data.Activity, timeStart time.Time) error {
			return binding.SetTimeStart(timeStart)
		},
	)
	a.timeStartLabel = widget.NewLabel("Start")
	a.timeStartValue = xwidget.NewDateTime(a.timeStartBinding)
	a.timeEndBinding = data.NewSubBindingImperative[data.Activity, database.Activity, time.Time](
		a.binding,
		func(activity database.Activity) (time.Time, error) {
			return activity.TimeEnd, nil
		},
		func(binding data.Activity, timeEnd time.Time) error {
			return binding.SetTimeEnd(timeEnd)
		},
	)
	a.timeEndLabel = widget.NewLabel("End")
	a.timeEndValue = xwidget.NewDateTime(a.timeEndBinding)

	a.container = container.New(layout.NewFormLayout(),
		a.idLabel, a.idValue,
		a.locationLabel, a.locationValue,
		a.timeStartLabel, a.timeStartValue,
		a.timeEndLabel, a.timeEndValue,
	)
	return a
}

func (a *Activity) MinSize() fyne.Size {
	// TODO: MinSize must be the minsize of the form layout or wtv
	return a.container.MinSize()
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
		a.timeStartValue.Disable()
		a.timeEndValue.Disable()
	} else {
		a.locationValue.Enable()
		a.timeStartValue.Enable()
		a.timeEndValue.Enable()
	}
}

func (a *Activity) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
