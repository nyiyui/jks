package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/jmoiron/sqlx"
	"nyiyui.ca/jks/data"
	"nyiyui.ca/jks/database"
)

type TaskInfo struct {
	widget.BaseWidget
	binding   data.GenericBinding[database.Task]
	task      database.Task
	container *fyne.Container

	idLabel *widget.Label
	idValue *widget.Label

	qtLabel *widget.Label
	qtValue *widget.Entry

	descView *widget.RichText
	descEdit *widget.Entry
}

func NewTaskInfo(binding data.GenericBinding[database.Task]) *TaskInfo {
	ti := new(TaskInfo)
	ti.binding = binding

	ti.idLabel = widget.NewLabel("ID")
	ti.idValue = widget.NewLabel("")

	ti.qtLabel = widget.NewLabel("Quick Title")
	ti.qtValue = widget.NewEntry()
	ti.qtValue.OnChanged = func(quickTitle string) {
		ti.task.QuickTitle = quickTitle
		err := ti.binding.Set(ti.task)
		if err != nil {
			fyne.LogError("set quick title failed", err)
			return
		}
	}
	ti.descView = widget.NewRichText()
	ti.descEdit = widget.NewMultiLineEntry()
	ti.descEdit.Wrapping = fyne.TextWrapWord
	ti.descEdit.OnChanged = ti.updateDescription

	ti.container = container.New(
		layout.NewVBoxLayout(),
		container.New(layout.NewFormLayout(),
			ti.idLabel, ti.idValue,
			ti.qtLabel, ti.qtValue,
		),
		container.New(layout.NewGridLayout(2),
			ti.descView,
			ti.descEdit,
		),
	)
	ti.binding.AddListener(ti)
	ti.ExtendBaseWidget(ti)
	return ti
}

func (ti *TaskInfo) updateDescription(s string) {
	ti.task.Description = s
	err := ti.binding.Set(ti.task)
	if err != nil {
		fyne.LogError("failed to set task", err)
		return
	}
}

func (ti *TaskInfo) DataChanged() {
	task, err := ti.binding.Get()
	if err != nil {
		fyne.LogError("failed to get task", err)
		return
	}
	if task == (database.Task{}) {
		return
	}
	ti.task = task
	ti.idValue.Text = fmt.Sprint(ti.task.ID)
	ti.idValue.Refresh()
	ti.qtValue.Text = ti.task.QuickTitle
	ti.qtValue.Refresh()
	ti.descView.ParseMarkdown(ti.task.Description)
	ti.descView.Refresh()
	ti.descEdit.Text = ti.task.Description
	ti.descEdit.Refresh()
}

func (ti *TaskInfo) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ti.container)
}

type AddTask struct {
	widget.BaseWidget

	db     *sqlx.DB
	window fyne.Window

	taskInfo *TaskInfo
	task     *data.LazyBinding[database.Task]
	rowid    int64

	container *fyne.Container
}

func NewAddTask(db *sqlx.DB, window fyne.Window) *AddTask {
	at := new(AddTask)
	at.ExtendBaseWidget(at)

	at.db = db
	at.window = window

	at.task = data.NewLazyBinding(
		func(saved database.Task) (data.GenericBinding[database.Task], error) {
			result, err := at.db.Exec(
				`INSERT INTO tasks (description, quick_title, deadline, due) VALUES (?, ?, ?, ?)`,
				saved.Description,
				saved.QuickTitle,
				toUnix(saved.Deadline),
				toUnix(saved.Due),
			)
			if err != nil {
				return nil, err
			}
			rowid, err := result.LastInsertId()
			if err != nil {
				return nil, err
			}
			at.rowid = rowid
			return data.NewTask(at.db, rowid), nil
		},
		database.Task{},
	)
	at.taskInfo = NewTaskInfo(at.task)

	at.container = container.NewStack(at.taskInfo)
	return at
}

func (at *AddTask) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(at.container)
}

func toUnix(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}

type AddTaskDialog struct {
	*dialog.CustomDialog
	at     *AddTask
	cancel *widget.Button
	ok     *widget.Button
}

func NewAddTaskDialog(title string, db *sqlx.DB, window fyne.Window) *AddTaskDialog {
	atd := new(AddTaskDialog)
	atd.at = NewAddTask(db, window)
	atd.cancel = widget.NewButton("Cancel", atd.onCancel)
	atd.ok = widget.NewButton("OK", atd.onOK)
	atd.ok.Importance = widget.HighImportance
	atd.CustomDialog = dialog.NewCustom(title, "Cancel", atd.at, window)
	atd.SetButtons([]fyne.CanvasObject{atd.ok, atd.cancel})
	atd.at.task.AddListener(atd)
	return atd
}

// implementation detail. Do not call.
func (atd *AddTaskDialog) DataChanged() {
	if atd.at.task.Initialized() {
		atd.ok.Enable()
	} else {
		atd.ok.Disable()
	}
}

func (atd *AddTaskDialog) onCancel() {
	if !atd.at.task.Initialized() {
		// nothing to do, since nothing saved
		return
	}
	_, err := atd.at.db.Exec(`DELETE FROM tasks WHERE id = ?`, atd.at.rowid)
	if err != nil {
		dialog.ShowError(fmt.Errorf("deleting task id %d on dialog cancel: %w", atd.at.rowid, err), atd.at.window)
		return
	}
}

func (atd *AddTaskDialog) onOK() {
	atd.CustomDialog.Hide()
}
