package ccsyntax

import (
	"fmt"
	"sync"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ConfigExecutionContext interface {
	GetName() string
	Add(oc *OriginContext) error
	AddBlock(oc *OriginContext) error
	GetDAG(oc *OriginContext) dag.DAG
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
	m              sync.RWMutex
	BlockDAGs      map[string]dag.DAG
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

// func (r *cfgExecContext) Add(fow FOW, gvk *schema.GroupVersionKind, rootVertexName string) error {
func (r *cfgExecContext) Add(oc *OriginContext) error {
	r.m.Lock()
	defer r.m.Unlock()
	switch oc.FOW {
	case FOWFor:
		// rootVertexName -> oc.VertexName
		r.For[*oc.GVK] = map[Operation]*DAGCtx{
			OperationApply: {
				DAG:            dag.New(),
				RootVertexName: oc.VertexName,
				BlockDAGs:      map[string]dag.DAG{},
			},
			OperationDelete: {
				DAG:            dag.New(),
				RootVertexName: oc.VertexName,
				BlockDAGs:      map[string]dag.DAG{},
			},
		}
	case FOWOwn:
		r.own[*oc.GVK] = map[Operation]*DAGCtx{}
	case FOWWatch:
		r.watch[*oc.GVK] = map[Operation]*DAGCtx{
			OperationApply: {
				DAG:            dag.New(),
				RootVertexName: oc.VertexName,
				BlockDAGs:      map[string]dag.DAG{},
			},
		}
	default:
		return fmt.Errorf("unknown FOW, got: %s", oc.FOW)
	}
	return nil
}

func (r *cfgExecContext) AddBlock(oc *OriginContext) error {
	dctx := r.GetDAGCtx(oc.FOW, oc.GVK, oc.Operation)
	if dctx == nil {
		return fmt.Errorf("dag context not found, got: %v", oc)
	}
	dctx.m.Lock()
	defer dctx.m.Unlock()
	dctx.BlockDAGs[oc.VertexName] = dag.New()
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

func (r *cfgExecContext) GetDAG(oc *OriginContext) dag.DAG {
	dctx := r.GetDAGCtx(oc.FOW, oc.GVK, oc.Operation)
	if dctx == nil {
		return nil
	}
	if oc.BlockIndex == 0 && oc.BlockVertexName == "" {
		return dctx.DAG
	}
	dctx.m.RLock()
	defer dctx.m.RUnlock()
	return dctx.BlockDAGs[oc.BlockVertexName]
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
