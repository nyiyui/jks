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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestCalendar(t *testing.T) {
	now := time.Now()
	c := newCalendarWithSetTime(now)
	assert.Equal(t, now.Day(), c.currentTime.Day())
	assert.Equal(t, int(now.Month()), int(c.currentTime.Month()))
	assert.Equal(t, now.Year(), c.currentTime.Year())

	_ = test.WidgetRenderer(c) // and render
	assert.Equal(t, now.Format("January 2006"), c.monthLabel.Text)
}

func TestCalendar_ButtonDate(t *testing.T) {
	date := time.Now()
	c := newCalendarWithSetTime(date)
	_ = test.WidgetRenderer(c) // and render

	endNextMonth := date.AddDate(0, 1, 0).AddDate(0, 0, -(date.Day() - 1))
	last := endNextMonth.AddDate(0, 0, -1)

	firstDate := firstDateButton(c.dates)
	assert.Equal(t, "1", firstDate.Text)
	lastDate := c.dates.Objects[len(c.dates.Objects)-1].(*widget.Button)
	assert.Equal(t, strconv.Itoa(last.Day()), lastDate.Text)
}

func TestCalendar_Next(t *testing.T) {
	date := time.Now()
	c := newCalendarWithSetTime(date)
	_ = test.WidgetRenderer(c) // and render

	assert.Equal(t, date.Format("January 2006"), c.monthLabel.Text)

	test.Tap(c.monthNext)
	date = date.AddDate(0, 1, 0)
	assert.Equal(t, date.Format("January 2006"), c.monthLabel.Text)
}

func TestCalendar_Previous(t *testing.T) {
	date := time.Now()
	c := newCalendarWithSetTime(date)
	_ = test.WidgetRenderer(c) // and render

	assert.Equal(t, date.Format("January 2006"), c.monthLabel.Text)

	test.Tap(c.monthPrevious)
	date = date.AddDate(0, -1, 0)
	assert.Equal(t, date.Format("January 2006"), c.monthLabel.Text)
}

func TestCalendar_Resize(t *testing.T) {
	date := time.Now()
	c := newCalendarWithSetTime(date)
	r := test.WidgetRenderer(c) // and render
	layout := c.dates.Layout.(*calendarLayout)

	baseSize := c.MinSize()
	r.Layout(baseSize)
	min := layout.cellSize

	r.Layout(baseSize.AddWidthHeight(100, 0))
	assert.Greater(t, layout.cellSize.Width, min.Width)
	assert.Equal(t, layout.cellSize.Height, min.Height)

	r.Layout(baseSize.AddWidthHeight(0, 100))
	assert.Equal(t, layout.cellSize.Width, min.Width)
	assert.Greater(t, layout.cellSize.Height, min.Height)

	r.Layout(baseSize.AddWidthHeight(100, 100))
	assert.Greater(t, layout.cellSize.Width, min.Width)
	assert.Greater(t, layout.cellSize.Height, min.Height)
}

func firstDateButton(c *fyne.Container) *widget.Button {
	for _, b := range c.Objects {
		if nonBlank, ok := b.(*widget.Button); ok {
			return nonBlank
		}
	}

	return nil
}
