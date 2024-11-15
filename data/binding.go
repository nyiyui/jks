package data

import (
	"fyne.io/fyne/v2/data/binding"
)

type GenericBinding[T any] interface {
	binding.DataItem
	Get() (T, error)
	Set(T) error
}

type subBinding[B GenericBinding[M], M, S any] struct {
	main     B
	compress func(S) (M, error)
	expand   func(M) (S, error)
	apply    func(B, S) error
}

func NewSubBinding[M, S any](
	main GenericBinding[M],
	expand func(M) (S, error),
	compress func(S) (M, error),
) GenericBinding[S] {
	return &subBinding[GenericBinding[M], M, S]{
		main,
		compress,
		expand,
		nil,
	}
}

func NewSubBindingImperative[B GenericBinding[M], M, S any](
	main B,
	expand func(M) (S, error),
	apply func(main B, value S) error,
) GenericBinding[S] {
	return &subBinding[B, M, S]{
		main,
		nil,
		expand,
		apply,
	}
}

func (sb *subBinding[B, M, S]) AddListener(dl binding.DataListener) {
	sb.main.AddListener(dl)
}

func (sb *subBinding[B, M, S]) RemoveListener(dl binding.DataListener) {
	sb.main.RemoveListener(dl)
}

func (sb *subBinding[B, M, S]) Get() (S, error) {
	m, err := sb.main.Get()
	if err != nil {
		var zero S
		return zero, err
	}
	return sb.expand(m)
}

func (sb *subBinding[B, M, S]) Set(value S) error {
	if sb.compress != nil {
		mainValue, err := sb.compress(value)
		if err != nil {
			return err
		}
		return sb.main.Set(mainValue)
	}
	if sb.apply != nil {
		return sb.apply(sb.main, value)
	}
	panic("unreachable")
}

type Typed[T any] struct {
	lower binding.Untyped
}

func NewTyped[T any]() *Typed[T] {
	return &Typed[T]{lower: binding.NewUntyped()}
}

func (t *Typed[T]) AddListener(dl binding.DataListener) {
	t.lower.AddListener(dl)
}

func (t *Typed[T]) RemoveListener(dl binding.DataListener) {
	t.lower.RemoveListener(dl)
}

func (t *Typed[T]) Get() (T, error) {
	value, err := t.lower.Get()
	if value == nil {
		var zero T
		return zero, err
	}
	return value.(T), err
}

func (t *Typed[T]) Set(value T) error {
	return t.lower.Set(value)
}

type proxy struct {
	f func()
}

func NewProxy(f func()) binding.DataListener {
	return &proxy{f}
}

func (p *proxy) DataChanged() {
	p.f()
}
