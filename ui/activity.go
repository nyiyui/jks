package ui

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	datepicker "github.com/sdassow/fyne-datepicker"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/data"
	"nyiyui.ca/jks/database"
)

var _ = datepicker.NewDatePicker

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
	activity        database.Activity
	idLabel         *widget.Label
	idValue         *widget.Label
	locationLabel   *widget.Label
	locationValue   *widget.Entry
	locationBinding data.GenericBinding[string]
	timeEndLabel    *widget.Label
	timeEndValue    *widget.Entry
	timeEndBinding  data.GenericBinding[string]
	timeEndSetNow   *widget.Button
	binding         data.Activity

	container *fyne.Container
}

func NewActivity(activityBinding data.Activity) *Activity {
	a := new(Activity)
	a.binding = activityBinding
	a.binding.AddListener(a)
	a.idLabel = widget.NewLabel("ID")
	a.idValue = widget.NewLabel("")
	a.locationBinding = data.NewSubBinding[database.Activity, string](
		a.binding,
		func(activity database.Activity) (string, error) {
			return activity.Location, nil
		},
		func(location string) (database.Activity, error) {
			activity, err := a.binding.Get()
			if err != nil {
				return database.Activity{}, err
			}
			activity.Location = location
			return activity, nil
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
	a.timeEndBinding = data.NewSubBinding[database.Activity, string](
		a.binding,
		func(activity database.Activity) (string, error) {
			return strconv.FormatInt(activity.TimeEnd.Unix(), 10), nil
		},
		func(timeEnd string) (database.Activity, error) {
			timeEnd2, err := strconv.ParseInt(timeEnd, 10, 64)
			if err != nil {
				return database.Activity{}, err
			}
			activity, err := a.binding.Get()
			if err != nil {
				return database.Activity{}, err
			}
			activity.TimeEnd = time.Unix(timeEnd2, 0)
			return activity, nil
		},
	)
	a.timeEndLabel = widget.NewLabel("End")
	a.timeEndValue = widget.NewEntry()
	a.timeEndValue.Bind(a.timeEndBinding)
	a.timeEndSetNow = widget.NewButton("Now", func() {
		// TODO: find a way to set time to now without resetting undo
		a.timeEndBinding.Set(strconv.FormatInt(time.Now().Unix(), 10))
	})
	a.timeEndValue.Validator = func(s string) error {
		_, err := strconv.ParseInt(s, 10, 32)
		return err
	}
	a.timeEndValue.OnChanged = func(newValue string) {
		epoch, err := strconv.ParseInt(newValue, 10, 32)
		if err != nil {
			panic(err) // should have already been validated
		}
		a.activity.TimeEnd = time.Unix(epoch, 0)
		err = a.binding.Set(a.activity)
		if err != nil {
			log.Printf("failed to set activity binding: %s", err)
		}
		return
	}

	a.container = container.New(layout.NewFormLayout(),
		a.idLabel, a.idValue,
		a.locationLabel, a.locationValue,
		a.timeEndLabel, container.NewHBox(a.timeEndValue, a.timeEndSetNow))
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
		a.timeEndValue.Disable()
		a.timeEndSetNow.Disable()
	} else {
		a.locationValue.Enable()
		a.timeEndValue.Enable()
		a.timeEndSetNow.Enable()
	}
}

func (a *Activity) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
