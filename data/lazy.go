package data

import (
	"time"

	"nyiyui.ca/jks/database"
)

type LazyBinding[T any] struct {
	*baseBinding
	lower    GenericBinding[T]
	NewLower func(data T) (GenericBinding[T], error)
	Initial  T
}

func NewLazyBinding[T any](newLower func(data T) (GenericBinding[T], error), initial T) *LazyBinding[T] {
	return &LazyBinding[T]{
		new(baseBinding),
		nil,
		newLower,
		initial,
	}
}

func (l *LazyBinding[T]) Get() (T, error) {
	if l.lower == nil {
		return l.Initial, nil
	}
	return l.lower.Get()
}

func (l *LazyBinding[T]) Set(data T) error {
	if l.lower == nil {
		var err error
		l.lower, err = l.NewLower(data)
		if err != nil {
			return err
		}
		l.notifyAllListeners()
		if l.lower == nil {
			panic("NewLower must return a non-nil GenericBinding unless an error")
		}
	}
	err := l.lower.Set(data)
	if err != nil {
		return err
	}
	l.notifyAllListeners()
	return nil
}

func (l *LazyBinding[T]) Initialized() bool {
	return l.lower != nil
}

type LazyTaskBinding struct {
	*LazyBinding[database.Task]
}

func (lt *LazyTaskBinding) SetRowid(int64) error { return nil }

func (lt *LazyTaskBinding) GetTotalTime() (time.Duration, error) {
	return 0, nil
}
