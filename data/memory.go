package data

import (
	"time"

	"nyiyui.ca/jks/database"
)

type activityMemory struct {
	*baseBinding
	rowid int64
	a     database.Activity
}

func NewActivityMemory(rowid int64, a database.Activity) Activity {
	return &activityMemory{new(baseBinding), rowid, a}
}

func (a *activityMemory) Get() (database.Activity, error) {
	return a.a, nil
}

func (a *activityMemory) Set(a2 database.Activity) error {
	a.a = a2
	a.notifyAllListeners()
	return nil
}

func (a *activityMemory) SetRowid(rowid int64) error {
	a.rowid = rowid
	a.notifyAllListeners()
	return nil
}

func (a *activityMemory) SetLocation(location string) error {
	a.a.Location = location
	a.notifyAllListeners()
	return nil
}

func (a *activityMemory) SetTimeStart(timeStart time.Time) error {
	a.a.TimeStart = timeStart
	a.notifyAllListeners()
	return nil
}

func (a *activityMemory) SetTimeEnd(timeEnd time.Time) error {
	a.a.TimeEnd = timeEnd
	a.notifyAllListeners()
	return nil
}
