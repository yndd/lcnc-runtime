package kv

import (
	"fmt"
	"sync"
)

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
		d[k] = copy(v)
	}
	return d
}

func copy(v any) any {
	switch v := v.(type) {
	case map[string]any:
		r := make(map[string]any, len(v))
		for k, vd := range v {
			r[k] = copy(vd)
		}
		return r
	case []any:
		r := make([]any, 0, len(v))
		for _, vd := range v {
			r = append(r, copy(vd))
		}
		return r
	default:
		fmt.Printf("%T\n", v)
		return v
	}
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
