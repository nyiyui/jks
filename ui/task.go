package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
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

	descValue *widget.RichText
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
	ti.descValue = widget.NewRichText()

	ti.container = container.New(
		layout.NewVBoxLayout(),
		container.New(layout.NewFormLayout(),
			ti.idLabel, ti.idValue,
			ti.qtLabel, ti.qtValue,
		),
		ti.descValue,
	)
	ti.binding.AddListener(ti)
	return ti
}

func (ti *TaskInfo) DataChanged() {
	task, err := ti.binding.Get()
	if err != nil {
		log.Printf("failed to get ID: %s", err)
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
	ti.descValue.ParseMarkdown(ti.task.Description)
	ti.descValue.Refresh()
}

func (ti *TaskInfo) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ti.container)
}
