package xwidget

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"nyiyui.ca/jks/data"
)

type DateTime struct {
	widget.DisableableWidget
	container      *fyne.Container
	dateEntry      *widget.Entry
	timeEntry      *widget.Entry
	dayOfWeekLabel *widget.Label
	nowButton      *widget.Button

	binding data.GenericBinding[time.Time]
}

func NewDateTime(binding data.GenericBinding[time.Time]) *DateTime {
	dt := &DateTime{
		binding:        binding,
		dateEntry:      widget.NewEntry(),
		timeEntry:      widget.NewEntry(),
		dayOfWeekLabel: widget.NewLabel(""),
	}
	dt.nowButton = widget.NewButton("Now", dt.setNow)
	dt.binding.AddListener(dt)
	dt.dateEntry.Wrapping = fyne.TextWrapOff
	dt.timeEntry.Wrapping = fyne.TextWrapOff
	dt.dateEntry.Validator = newTimeValidator("2006-01-02")
	dt.timeEntry.Validator = newTimeValidator("15:04")
	dt.dateEntry.OnChanged = func(s string) {
		tDelta, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			return
		}
		t, err := dt.binding.Get()
		if err != nil {
			fyne.LogError("get time to update time", err)
		}
		t1 := time.Date(tDelta.Year(), tDelta.Month(), tDelta.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
		err = dt.binding.Set(t1)
		if err != nil {
			fyne.LogError("update time", err)
		}
	}
	dt.timeEntry.OnChanged = func(s string) {
		tDelta, err := time.ParseInLocation("15:04", s, time.Local)
		if err != nil {
			return
		}
		t, err := dt.binding.Get()
		if err != nil {
			fyne.LogError("get time to update time", err)
		}
		t1 := time.Date(t.Year(), t.Month(), t.Day(), tDelta.Hour(), tDelta.Minute(), tDelta.Second(), t.Nanosecond(), t.Location())
		err = dt.binding.Set(t1)
		if err != nil {
			fyne.LogError("update time", err)
		}
	}
	dt.ExtendBaseWidget(dt)
	dt.container = container.New(layout.NewGridLayout(4), dt.dateEntry, dt.dayOfWeekLabel, dt.timeEntry, dt.nowButton)
	return dt
}

func (dt *DateTime) setNow() {
	t1 := time.Now()
	err := dt.binding.Set(t1)
	if err != nil {
		fyne.LogError("update time", err)
	}
}

func newTimeValidator(layout string) func(string) error {
	return func(s string) error {
		_, err := time.Parse(layout, s)
		return err
	}
}

func (dt *DateTime) DataChanged() {
	currentTime, err := dt.binding.Get()
	if err != nil {
		fyne.LogError("failed to get bound time", err)
		return
	}
	dt.dateEntry.Text = currentTime.Local().Format("2006-01-02")
	dt.timeEntry.Text = currentTime.Local().Format("15:04")
	dt.dayOfWeekLabel.Text = dayOfWeek(currentTime.Local())
	dt.Refresh()
}

func (dt *DateTime) Refresh() {
	dt.dateEntry.Refresh()
	dt.timeEntry.Refresh()
	dt.dayOfWeekLabel.Refresh()
	dt.BaseWidget.Refresh()
}

func (dt *DateTime) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(dt.container)
}

func (dt *DateTime) Disable() {
	dt.dateEntry.Disable()
	dt.timeEntry.Disable()
	dt.nowButton.Disable()
}

func (dt *DateTime) Enable() {
	dt.dateEntry.Enable()
	dt.timeEntry.Enable()
	dt.nowButton.Enable()
}

func dayOfWeek(t time.Time) string {
	return t.Weekday().String()
}
