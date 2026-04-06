package ui

const (
	defaultFocusCount = 1
	defaultFocusIndex = 0
)

// FocusRing manages linear focus over N focusable elements.
// It wraps on next/prev and can be reused by any TUI model.
type FocusRing struct {
	count int
	index int
}

func NewFocusRing(count int) FocusRing {
	r := FocusRing{}
	r.SetCount(count)
	return r
}

func (r *FocusRing) SetCount(count int) {
	if count <= 0 {
		count = defaultFocusCount
	}
	r.count = count
	if r.index >= r.count || r.index < 0 {
		r.index = defaultFocusIndex
	}
}

func (r *FocusRing) Focus(index int) {
	if r.count <= 0 {
		r.SetCount(defaultFocusCount)
	}
	if index < 0 {
		index = 0
	}
	if index >= r.count {
		index = r.count - 1
	}
	r.index = index
}

func (r *FocusRing) Next() int {
	if r.count <= 0 {
		r.SetCount(defaultFocusCount)
	}
	r.index = (r.index + 1) % r.count
	return r.index
}

func (r *FocusRing) Prev() int {
	if r.count <= 0 {
		r.SetCount(defaultFocusCount)
	}
	r.index--
	if r.index < 0 {
		r.index = r.count - 1
	}
	return r.index
}

func (r FocusRing) Current() int {
	return r.index
}

func (r FocusRing) Is(index int) bool {
	return r.index == index
}
