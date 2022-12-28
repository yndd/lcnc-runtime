package kv

import "sync"

type RecordKVFn func(KV)

type KV interface {
	AddEntry(k string, v any)
	Add(KV)
	Get() map[string]any
	GetValue(string) any
	Length() int
}

func New() KV {
	return &kv{
		d: map[string]any{},
	}
}

type kv struct {
	m sync.RWMutex
	d map[string]any
}

func (r *kv) AddEntry(k string, v any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.d[k] = v
}

func (r *kv) Add(o KV) {
	r.m.Lock()
	defer r.m.Unlock()
	for k, v := range o.Get() {
		r.d[k] = v
	}
}

func (r *kv) Get() map[string]any {
	r.m.RLock()
	defer r.m.RUnlock()
	d := make(map[string]any, len(r.d))
	for k, v := range r.d {
		d[k] = v
	}
	return d
}

func (r *kv) GetValue(k string) any {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.d[k]
}

func (r *kv) Length() int {
	r.m.RLock()
	defer r.m.RUnlock()
	return len(r.d)
}
