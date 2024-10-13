// Package xwidget was copied from https://github.com/fyne-io/fyne-x at commit 214f23daf11676c42f114c47aba83a9c17e08b6d
package xwidget

// This file was copied and modified from:
// https://github.com/fyne-io/fyne-x/blob/8b5b5bfe65efaa90f4a8bde186d027bf20805112/widget/calendar.go
/*
BSD 3-Clause License

Copyright (C) 2020 Fyne.io developers and community (see AUTHORS)
All rights reserved.


Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:
    * Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.
    * Neither the name of Fyne.io nor the names of its contributors may be
      used to endorse or promote products derived from this software without
      specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"math"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"nyiyui.ca/jks/data"
)

// Declare conformity with Layout interface
var _ fyne.Layout = (*calendarLayout)(nil)

const (
	daysPerWeek      = 7
	maxWeeksPerMonth = 6
)

type calendarLayout struct {
	cellSize fyne.Size
}

func newCalendarLayout() fyne.Layout {
	return &calendarLayout{}
}

// Get the leading edge position of a grid cell.
// The row and col specify where the cell is in the calendar.
func (g *calendarLayout) getLeading(row, col int) fyne.Position {
	x := (g.cellSize.Width) * float32(col)
	y := (g.cellSize.Height) * float32(row)

	return fyne.NewPos(float32(math.Round(float64(x))), float32(math.Round(float64(y))))
}

// Get the trailing edge position of a grid cell.
// The row and col specify where the cell is in the calendar.
func (g *calendarLayout) getTrailing(row, col int) fyne.Position {
	return g.getLeading(row+1, col+1)
}

// Layout is called to pack all child objects into a specified size.
// For a GridLayout this will pack objects into a table format with the number
// of columns specified in our constructor.
func (g *calendarLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	weeks := 1
	day := 0
	for i, child := range objects {
		if !child.Visible() {
			continue
		}

		if day%daysPerWeek == 0 && i >= daysPerWeek {
			weeks++
		}
		day++
	}

	g.cellSize = fyne.NewSize(size.Width/float32(daysPerWeek),
		size.Height/float32(weeks))
	row, col := 0, 0
	i := 0
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		lead := g.getLeading(row, col)
		trail := g.getTrailing(row, col)
		child.Move(lead)
		child.Resize(fyne.NewSize(trail.X, trail.Y).Subtract(lead))

		if (i+1)%daysPerWeek == 0 {
			row++
			col = 0
		} else {
			col++
		}
		i++
	}
}

// MinSize sets the minimum size for the calendar
func (g *calendarLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	pad := theme.Padding()
	largestMin := widget.NewLabel("22").MinSize()
	return fyne.NewSize(largestMin.Width*daysPerWeek+pad*(daysPerWeek-1),
		largestMin.Height*maxWeeksPerMonth+pad*(maxWeeksPerMonth-1))
}

// Calendar creates a new date time picker which returns a time object
type Calendar struct {
	widget.DisableableWidget
	currentTime  time.Time
	timeData     data.GenericBinding[time.Time]
	dataListener *dataListener

	monthPrevious *widget.Button
	monthNext     *widget.Button
	monthCurrent  *widget.Button
	monthLabel    *widget.Label

	dates *fyne.Container
}

func (c *Calendar) daysOfMonth() []fyne.CanvasObject {
	start := time.Date(c.currentTime.Year(), c.currentTime.Month(), 1, 0, 0, 0, 0, c.currentTime.Location())
	buttons := []fyne.CanvasObject{}

	//account for Go time pkg starting on sunday at index 0
	dayIndex := int(start.Weekday())
	if dayIndex == 0 {
		dayIndex += daysPerWeek
	}

	//add spacers if week doesn't start on Monday
	for i := 0; i < dayIndex-1; i++ {
		buttons = append(buttons, layout.NewSpacer())
	}

	for d := start; d.Month() == start.Month(); d = d.AddDate(0, 0, 1) {
		dayNum := d.Day()
		s := strconv.Itoa(dayNum)
		b := widget.NewButton(s, func() {
			selectedDate := dateForButton(c.currentTime, dayNum)
			err := c.timeData.Set(selectedDate)
			if err != nil {
				fyne.LogError("time data couldn't be set", err)
				return
			}
		})
		b.Importance = widget.LowImportance

		buttons = append(buttons, b)
	}

	return buttons
}

func dateForButton(currentTime time.Time, dayNum int) time.Time {
	oldName, off := currentTime.Zone()
	return time.Date(currentTime.Year(), currentTime.Month(), dayNum, currentTime.Hour(), currentTime.Minute(), 0, 0, time.FixedZone(oldName, off)).In(currentTime.Location())
}

func (c *Calendar) monthYear() string {
	return c.currentTime.Format("January 2006")
}

func (c *Calendar) calendarObjects() []fyne.CanvasObject {
	columnHeadings := []fyne.CanvasObject{}
	for i := 0; i < daysPerWeek; i++ {
		j := i + 1
		if j == daysPerWeek {
			j = 0
		}

		t := widget.NewLabel(strings.ToUpper(time.Weekday(j).String()[:3]))
		t.Alignment = fyne.TextAlignCenter
		columnHeadings = append(columnHeadings, t)
	}
	columnHeadings = append(columnHeadings, c.daysOfMonth()...)

	return columnHeadings
}

// CreateRenderer returns a new WidgetRenderer for this widget.
// This should not be called by regular code, it is used internally to render a widget.
func (c *Calendar) CreateRenderer() fyne.WidgetRenderer {
	c.monthPrevious = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		c.currentTime = c.currentTime.AddDate(0, -1, 0)
		// Dates are 'normalised', forcing date to start from the start of the month ensures move from March to February
		c.currentTime = time.Date(c.currentTime.Year(), c.currentTime.Month(), 1, 0, 0, 0, 0, c.currentTime.Location())
		c.monthLabel.SetText(c.monthYear())
		c.dates.Objects = c.calendarObjects()
	})
	c.monthPrevious.Importance = widget.LowImportance

	c.monthCurrent = widget.NewButton("Now", func() {
		c.currentTime = time.Now()
		c.monthLabel.SetText(c.monthYear())
		c.dates.Objects = c.calendarObjects()
	})
	c.monthCurrent.Importance = widget.LowImportance

	c.monthNext = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		c.currentTime = c.currentTime.AddDate(0, 1, 0)
		c.monthLabel.SetText(c.monthYear())
		c.dates.Objects = c.calendarObjects()
	})
	c.monthNext.Importance = widget.LowImportance

	c.monthLabel = widget.NewLabel(c.monthYear())

	nextGroup := container.NewHBox(c.monthCurrent, c.monthNext)
	nav := container.New(layout.NewBorderLayout(nil, nil, c.monthPrevious, nextGroup),
		c.monthPrevious, nextGroup, container.NewCenter(c.monthLabel))

	c.dates = container.New(newCalendarLayout(), c.calendarObjects()...)

	dateContainer := container.NewBorder(nav, nil, nil, nil, c.dates)

	return widget.NewSimpleRenderer(dateContainer)
}

func (c *Calendar) dataChanged() {
	var err error
	c.currentTime, err = c.timeData.Get()
	if err != nil {
		fyne.LogError("failed to get current time from binding", err)
	}
	if c.monthLabel != nil && c.dates != nil {
		c.monthLabel.SetText(c.monthYear())
		c.dates.Objects = c.calendarObjects()
	}
	c.Refresh()
}

func (c *Calendar) Enable() {
	if c.monthPrevious != nil {
		c.monthPrevious.Enable()
	}
	if c.monthCurrent != nil {
		c.monthCurrent.Enable()
	}
	if c.monthNext != nil {
		c.monthNext.Enable()
	}
	if c.dates != nil {
		for _, obj := range c.dates.Objects {
			if btn, ok := obj.(*widget.Button); ok {
				btn.Enable()
			}
		}
	}
	c.DisableableWidget.Enable()
	c.Refresh()
}

func (c *Calendar) Disable() {
	if c.monthPrevious != nil {
		c.monthPrevious.Disable()
	}
	if c.monthCurrent != nil {
		c.monthCurrent.Disable()
	}
	if c.monthNext != nil {
		c.monthNext.Disable()
	}
	if c.dates != nil {
		for _, obj := range c.dates.Objects {
			if btn, ok := obj.(*widget.Button); ok {
				btn.Disable()
			}
		}
	}
	c.DisableableWidget.Disable()
	c.Refresh()
}

// NewCalendar creates a calendar instance
func NewCalendar(timeData data.GenericBinding[time.Time]) *Calendar {
	c := &Calendar{
		timeData: timeData,
	}
	c.dataListener = &dataListener{c.dataChanged}
	c.dataChanged()
	timeData.AddListener(c.dataListener)
	c.ExtendBaseWidget(c)
	return c
}

func newCalendarWithSetTime(currentTime time.Time) *Calendar {
	c := &Calendar{
		currentTime: currentTime,
	}
	c.ExtendBaseWidget(c)
	return c
}

type dataListener struct {
	callback func()
}

func (d *dataListener) DataChanged() {
	d.callback()
}
