package lib

import "io"

type MyType struct{}

func (m MyType) InternalOnly() string { return "hello" }

func (m MyType) ExternallyUsed() string { return "world" }

func (m *MyType) Write(p []byte) (int, error) {
	return len(p), nil
}

var _ io.Writer = (*MyType)(nil)

type hidden struct{}

func (h hidden) Visible() string { return "" }

type Container[T any] struct{ val T }

func (c Container[T]) Get() T { return c.val }

func internalUse() {
	t := MyType{}
	_ = t.InternalOnly()
	_ = t.ExternallyUsed()
	_, _ = t.Write(nil)

	h := hidden{}
	_ = h.Visible()

	c := Container[int]{val: 42}
	_ = c.Get()
}
