package ccsyntax

import (
	"fmt"
	"sync"

	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ConfigExecutionContext interface {
	GetName() string
	Add(oc *OriginContext) error
	AddBlock(oc *OriginContext) error
	GetDAG(oc *OriginContext) rtdag.RuntimeDAG
	GetDAGCtx(fow FOW, gvk *schema.GroupVersionKind, op Operation) *RTDAGCtx
	GetFOW(fow FOW) map[schema.GroupVersionKind]OperationCtx
	GetForGVK() *schema.GroupVersionKind
	Print()
}

type cfgExecContext struct {
	name  string
	m     sync.RWMutex
	For   map[schema.GroupVersionKind]OperationCtx
	own   map[schema.GroupVersionKind]OperationCtx
	watch map[schema.GroupVersionKind]OperationCtx
}

type OperationCtx map[Operation]*RTDAGCtx

type RTDAGCtx struct {
	DAG            rtdag.RuntimeDAG
	RootVertexName string
	m              sync.RWMutex
	BlockDAGs      map[string]rtdag.RuntimeDAG
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
		r.For[*oc.GVK] = map[Operation]*RTDAGCtx{
			OperationApply: {
				DAG:            rtdag.New(),
				RootVertexName: oc.VertexName,
				BlockDAGs:      map[string]rtdag.RuntimeDAG{},
			},
			OperationDelete: {
				DAG:            rtdag.New(),
				RootVertexName: oc.VertexName,
				BlockDAGs:      map[string]rtdag.RuntimeDAG{},
			},
		}
	case FOWOwn:
		r.own[*oc.GVK] = map[Operation]*RTDAGCtx{}
	case FOWWatch:
		r.watch[*oc.GVK] = map[Operation]*RTDAGCtx{
			OperationApply: {
				DAG:            rtdag.New(),
				RootVertexName: oc.VertexName,
				BlockDAGs:      map[string]rtdag.RuntimeDAG{},
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
	dctx.BlockDAGs[oc.VertexName] = rtdag.New()
	return nil
}

func (r *cfgExecContext) GetDAGCtx(fow FOW, gvk *schema.GroupVersionKind, op Operation) *RTDAGCtx {
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

func (r *cfgExecContext) GetDAG(oc *OriginContext) rtdag.RuntimeDAG {
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
			gvkDAGMap[gvk] = map[Operation]*RTDAGCtx{}
			for op, d := range od {
				gvkDAGMap[gvk][op] = d
			}
		}
	case FOWOwn:
		for gvk, od := range r.own {
			gvkDAGMap[gvk] = map[Operation]*RTDAGCtx{}
			for op, d := range od {
				gvkDAGMap[gvk][op] = d
			}
		}
	case FOWWatch:
		for gvk, od := range r.watch {
			gvkDAGMap[gvk] = map[Operation]*RTDAGCtx{}
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

func (r *cfgExecContext) Print() {
	r.m.RLock()
	defer r.m.RUnlock()
	fmt.Printf("###### CEC #######\n")
	for gvk, oc := range r.For {
		fmt.Printf("gvk: %v\n", gvk)

		for op, dctx := range oc {
			fmt.Printf("  op: %s, RootVertexName: %s, blockDAGs: %d\n", op, dctx.RootVertexName, len(dctx.BlockDAGs))
			dctx.DAG.PrintVertices()
			for rootVertexName, d := range dctx.BlockDAGs {
				fmt.Printf("!!!!!!! block dag start: vertexName: %s, %s !!!!!!!!!!\n", rootVertexName, d.GetRootVertex())
				d.PrintVertices()
				fmt.Printf("!!!!!!! block dag stop : vertexName: %s, %s !!!!!!!!!!\n", rootVertexName, d.GetRootVertex())
			}
		}
	}
}
