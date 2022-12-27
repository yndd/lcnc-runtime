package input

import "sync"

type Input interface {
	Add(s string, v any)
	Get() map[string]any
	GetValue(s string) any
	Length() int
}

func New() Input {
	return &input{
		i: map[string]any{},
	}
}

type input struct {
	m sync.RWMutex
	i map[string]any
}

func (r *input) Add(s string, v any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.i[s] = v
}

func (r *input) Get() map[string]any {
	r.m.RLock()
	defer r.m.RUnlock()
	input := map[string]any{}
	for k, v := range r.i {
		input[k] = v
	}
	return input
}

func (r *input) GetValue(s string) any {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.i[s]
}

func (r *input) Length() int {
	r.m.RLock()
	defer r.m.RUnlock()
	return len(r.i)
}
