package ccsyntax

import (
	"fmt"
	"sync"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ConfigExecutionContext interface {
	GetName() string
	Add(fow FOW, gvk *schema.GroupVersionKind, rootVertexName string) error
	GetDAGCtx(fow FOW, gvk *schema.GroupVersionKind, op Operation) *DAGCtx
	GetFOW(fow FOW) map[schema.GroupVersionKind]OperationCtx
	GetForGVK() *schema.GroupVersionKind
}

type cfgExecContext struct {
	name  string
	m     sync.RWMutex
	For   map[schema.GroupVersionKind]OperationCtx
	own   map[schema.GroupVersionKind]OperationCtx
	watch map[schema.GroupVersionKind]OperationCtx
}

type OperationCtx map[Operation]*DAGCtx

type DAGCtx struct {
	DAG            dag.DAG
	RootVertexName string
}

func NewConfigExecutionContext(n string) ConfigExecutionContext {
	return &cfgExecContext{
		name:  n,
		For:   make(map[schema.GroupVersionKind]OperationCtx),
		own:   make(map[schema.GroupVersionKind]OperationCtx),
		watch: make(map[schema.GroupVersionKind]OperationCtx),
	}
}

func (r *cfgExecContext) GetName() string {
	return r.name
}

func (r *cfgExecContext) Add(fow FOW, gvk *schema.GroupVersionKind, rootVertexName string) error {
	r.m.Lock()
	defer r.m.Unlock()
	switch fow {
	case FOWFor:
		r.For[*gvk] = map[Operation]*DAGCtx{
			OperationApply:  {DAG: dag.New(), RootVertexName: rootVertexName},
			OperationDelete: {DAG: dag.New(), RootVertexName: rootVertexName},
		}
	case FOWOwn:
		r.own[*gvk] = map[Operation]*DAGCtx{}
	case FOWWatch:
		r.watch[*gvk] = map[Operation]*DAGCtx{
			OperationApply: {DAG: dag.New(), RootVertexName: rootVertexName},
		}
	default:
		return fmt.Errorf("unknown FOW, got: %s", fow)
	}
	return nil
}

func (r *cfgExecContext) GetDAGCtx(fow FOW, gvk *schema.GroupVersionKind, op Operation) *DAGCtx {
	r.m.RLock()
	defer r.m.RUnlock()
	switch fow {
	case FOWFor:
		od, ok := r.For[*gvk]
		if !ok {
			return nil
		}
		dctx, ok := od[op]
		if !ok {
			return nil
		}
		return dctx
	case FOWOwn:
		od, ok := r.own[*gvk]
		if !ok {
			return nil
		}
		dctx, ok := od[op]
		if !ok {
			return nil
		}
		return dctx
	case FOWWatch:
		od, ok := r.watch[*gvk]
		if !ok {
			return nil
		}
		dctx, ok := od[op]
		if !ok {
			return nil
		}
		return dctx
	}
	return nil
}

func (r *cfgExecContext) GetFOW(fow FOW) map[schema.GroupVersionKind]OperationCtx {
	// A copy is returned
	gvkDAGMap := map[schema.GroupVersionKind]OperationCtx{}
	r.m.RLock()
	defer r.m.RUnlock()
	switch fow {
	case FOWFor:
		for gvk, od := range r.For {
			gvkDAGMap[gvk] = map[Operation]*DAGCtx{}
			for op, d := range od {
				gvkDAGMap[gvk][op] = d
			}
		}
	case FOWOwn:
		for gvk, od := range r.own {
			gvkDAGMap[gvk] = map[Operation]*DAGCtx{}
			for op, d := range od {
				gvkDAGMap[gvk][op] = d
			}
		}
	case FOWWatch:
		for gvk, od := range r.watch {
			gvkDAGMap[gvk] = map[Operation]*DAGCtx{}
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
