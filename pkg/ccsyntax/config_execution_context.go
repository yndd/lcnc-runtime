package ccsyntax

import (
	"fmt"
	"sync"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ConfigExecutionContext interface {
	GetName() string
	Add(fow FOW, gvk *schema.GroupVersionKind) error
	GetDAG(fow FOW, gvk *schema.GroupVersionKind, op Operation) dag.DAG
	GetFOW(fow FOW) map[schema.GroupVersionKind]OperationDAG
	GetForGVK() *schema.GroupVersionKind
}

type cfgExecContext struct {
	name  string
	m     sync.RWMutex
	For   map[schema.GroupVersionKind]OperationDAG
	own   map[schema.GroupVersionKind]OperationDAG
	watch map[schema.GroupVersionKind]OperationDAG
}

type OperationDAG map[Operation]dag.DAG

func NewConfigExecutionContext(n string) ConfigExecutionContext {
	return &cfgExecContext{
		name:  n,
		For:   make(map[schema.GroupVersionKind]OperationDAG),
		own:   make(map[schema.GroupVersionKind]OperationDAG),
		watch: make(map[schema.GroupVersionKind]OperationDAG),
	}
}

func (r *cfgExecContext) GetName() string {
	return r.name
}

func (r *cfgExecContext) Add(fow FOW, gvk *schema.GroupVersionKind) error {
	r.m.Lock()
	defer r.m.Unlock()
	switch fow {
	case FOWFor:
		r.For[*gvk] = map[Operation]dag.DAG{
			OperationApply:  dag.New(),
			OperationDelete: dag.New(),
		}
	case FOWOwn:
		r.own[*gvk] = map[Operation]dag.DAG{}
	case FOWWatch:
		r.watch[*gvk] = map[Operation]dag.DAG{
			OperationApply: dag.New(),
		}
	default:
		return fmt.Errorf("unknown FOW, got: %s", fow)
	}
	return nil
}

func (r *cfgExecContext) GetDAG(fow FOW, gvk *schema.GroupVersionKind, op Operation) dag.DAG {
	r.m.RLock()
	defer r.m.RUnlock()
	switch fow {
	case FOWFor:
		od, ok := r.For[*gvk]
		if !ok {
			return nil
		}
		d, ok := od[op]
		if !ok {
			return nil
		}
		return d
	case FOWOwn:
		od, ok := r.own[*gvk]
		if !ok {
			return nil
		}
		d, ok := od[op]
		if !ok {
			return nil
		}
		return d
	case FOWWatch:
		od, ok := r.watch[*gvk]
		if !ok {
			return nil
		}
		d, ok := od[op]
		if !ok {
			return nil
		}
		return d
	}
	return nil
}

func (r *cfgExecContext) GetFOW(fow FOW) map[schema.GroupVersionKind]OperationDAG {
	// A copy is returned
	gvkDAGMap := map[schema.GroupVersionKind]OperationDAG{}
	r.m.RLock()
	defer r.m.RUnlock()
	switch fow {
	case FOWFor:
		for gvk, od := range r.For {
			gvkDAGMap[gvk] = map[Operation]dag.DAG{}
			for op, d := range od {
				gvkDAGMap[gvk][op] = d
			}
		}
	case FOWOwn:
		for gvk, od := range r.own {
			gvkDAGMap[gvk] = map[Operation]dag.DAG{}
			for op, d := range od {
				gvkDAGMap[gvk][op] = d
			}
		}
	case FOWWatch:
		for gvk, od := range r.watch {
			gvkDAGMap[gvk] = map[Operation]dag.DAG{}
			for op, d := range od {
				gvkDAGMap[gvk][op] = d
			}
		}
	}
	return gvkDAGMap
}

func (r *cfgExecContext) GetForGVK() *schema.GroupVersionKind {
	for gvk := range r.For {
		return &gvk
	}
	return &schema.GroupVersionKind{}
}
