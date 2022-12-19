package ccsyntax

import (
	"fmt"
	"sync"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ConfigExecutionContext interface {
	GetName() string
	Add(fow FOW, gvk schema.GroupVersionKind, d dag.DAG) error
	GetDAG(fow FOW, gvk schema.GroupVersionKind) dag.DAG
	GetFOW(fow FOW) map[schema.GroupVersionKind]dag.DAG
	GetForGVK() schema.GroupVersionKind
}

type cfgExecContext struct {
	name  string
	m     sync.RWMutex
	For   map[schema.GroupVersionKind]dag.DAG
	own   map[schema.GroupVersionKind]dag.DAG
	watch map[schema.GroupVersionKind]dag.DAG
}

func NewConfigExecutionContext(n string) ConfigExecutionContext {
	return &cfgExecContext{
		name:  n,
		For:   make(map[schema.GroupVersionKind]dag.DAG),
		own:   make(map[schema.GroupVersionKind]dag.DAG),
		watch: make(map[schema.GroupVersionKind]dag.DAG),
	}
}

func (r *cfgExecContext) GetName() string {
	return r.name
}

func (r *cfgExecContext) Add(fow FOW, gvk schema.GroupVersionKind, d dag.DAG) error {
	r.m.Lock()
	defer r.m.Unlock()
	switch fow {
	case FOWFor:
		r.For[gvk] = d
	case FOWOwn:
		r.own[gvk] = nil // own does not have a DAG so we initialize always to nil
	case FOWWatch:
		r.watch[gvk] = d
	default:
		return fmt.Errorf("unknown FOW, got: %s", fow)
	}
	return nil
}

func (r *cfgExecContext) GetDAG(fow FOW, gvk schema.GroupVersionKind) dag.DAG {
	r.m.RLock()
	defer r.m.RUnlock()
	switch fow {
	case FOWFor:
		return r.For[gvk]
	case FOWOwn:
		return r.own[gvk]
	case FOWWatch:
		return r.watch[gvk]
	}
	return nil
}

func (r *cfgExecContext) GetFOW(fow FOW) map[schema.GroupVersionKind]dag.DAG {
	// TODO make a copy
	r.m.RLock()
	defer r.m.RUnlock()
	switch fow {
	case FOWFor:
		return r.For
	case FOWOwn:
		return r.own
	case FOWWatch:
		return r.watch
	}
	return map[schema.GroupVersionKind]dag.DAG{}
}

func (r *cfgExecContext) GetForGVK() schema.GroupVersionKind {
	for gvk := range r.For {
		return gvk
	}
	return schema.GroupVersionKind{}
}
