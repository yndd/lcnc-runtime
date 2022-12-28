package slice

import "sync"

type RecordSliceEntryFn func(any)

type Slice interface {
	Add(v any)
	Get() []any
	Length() int
}

func New() Slice {
	return &slice{
		d: make([]any, 0),
	}
}

type slice struct {
	m sync.RWMutex
	d []any
}

func (r *slice) Add(v any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.d = append(r.d, v)
}

func (r *slice) Get() []any {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.d
}

func (r *slice) Length() int {
	r.m.RLock()
	defer r.m.RUnlock()
	return len(r.d)
}
