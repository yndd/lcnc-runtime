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
	GetDAG(fow FOW, gvkop GVKOperation) dag.DAG
	GetFOW(fow FOW) map[GVKOperation]dag.DAG
	GetForGVK() *schema.GroupVersionKind
}

type cfgExecContext struct {
	name  string
	m     sync.RWMutex
	For   map[GVKOperation]dag.DAG
	own   map[GVKOperation]dag.DAG
	watch map[GVKOperation]dag.DAG
}

type GVKOperation struct {
	GVK       schema.GroupVersionKind
	Operation Operation
}

func NewConfigExecutionContext(n string) ConfigExecutionContext {
	return &cfgExecContext{
		name:  n,
		For:   make(map[GVKOperation]dag.DAG),
		own:   make(map[GVKOperation]dag.DAG),
		watch: make(map[GVKOperation]dag.DAG),
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
		r.For[GVKOperation{GVK: *gvk, Operation: OperationApply}] = dag.New()
		r.For[GVKOperation{GVK: *gvk, Operation: OperationDelete}] = dag.New()
	case FOWOwn:
		r.own[GVKOperation{GVK: *gvk, Operation: OperationApply}] = nil
		r.own[GVKOperation{GVK: *gvk, Operation: OperationDelete}] = nil
	case FOWWatch:
		r.watch[GVKOperation{GVK: *gvk, Operation: OperationApply}] = dag.New()
		r.watch[GVKOperation{GVK: *gvk, Operation: OperationDelete}] = nil
	default:
		return fmt.Errorf("unknown FOW, got: %s", fow)
	}
	return nil
}

func (r *cfgExecContext) GetDAG(fow FOW, gvkop GVKOperation) dag.DAG {
	r.m.RLock()
	defer r.m.RUnlock()
	switch fow {
	case FOWFor:
		return r.For[gvkop]
	case FOWOwn:
		return r.own[gvkop]
	case FOWWatch:
		return r.watch[gvkop]
	}
	return nil
}

func (r *cfgExecContext) GetFOW(fow FOW) map[GVKOperation]dag.DAG {
	gvkDAGMap := map[GVKOperation]dag.DAG{}
	r.m.RLock()
	defer r.m.RUnlock()
	switch fow {
	case FOWFor:
		for gvk, d := range r.For {
			gvkDAGMap[gvk] = d
		}
	case FOWOwn:
		for gvk, d := range r.own {
			gvkDAGMap[gvk] = d
		}
	case FOWWatch:
		for gvk, d := range r.watch {
			gvkDAGMap[gvk] = d
		}
	}
	return gvkDAGMap
}

func (r *cfgExecContext) GetForGVK() *schema.GroupVersionKind {
	for gvkop := range r.For {
		return &gvkop.GVK
	}
	return &schema.GroupVersionKind{}
}
