package data

import "fyne.io/fyne/v2/data/binding"

type GenericBinding[T any] interface {
	binding.DataItem
	Get() (T, error)
	Set(T) error
}

type subBinding[M, S any] struct {
	main     GenericBinding[M]
	compress func(S) (M, error)
	expand   func(M) (S, error)
}

func NewSubBinding[M, S any](
	main GenericBinding[M],
	expand func(M) (S, error),
	compress func(S) (M, error),
) GenericBinding[S] {
	return &subBinding[M, S]{
		main,
		compress,
		expand,
	}
}

func (sb *subBinding[M, S]) AddListener(dl binding.DataListener) {
	sb.main.AddListener(dl)
}

func (sb *subBinding[M, S]) RemoveListener(dl binding.DataListener) {
	sb.main.RemoveListener(dl)
}

func (sb *subBinding[M, S]) Get() (S, error) {
	m, err := sb.main.Get()
	if err != nil {
		var zero S
		return zero, err
	}
	return sb.expand(m)
}

func (sb *subBinding[M, S]) Set(value S) error {
	mainValue, err := sb.compress(value)
	if err != nil {
		return err
	}
	return sb.main.Set(mainValue)
}
