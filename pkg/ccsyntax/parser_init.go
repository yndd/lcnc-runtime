package ccsyntax

import (
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) init() (ConfigExecutionContext, GlobalVariable, []Result) {
	i := initializer{
		cec:  NewConfigExecutionContext(r.cCfg.GetName()),
		gvar: NewGlobalVariable(r.cCfg.GetName()),
	}

	fnc := &WalkConfig{
		gvkObjectFn:     i.initGvk,
		functionBlockFn: i.initFunctionBlock,
	}
	// walk the config initialaizes the config execution context
	r.walkLcncConfig(fnc)

	return i.cec, i.gvar, i.result

}

type initializer struct {
	cec    ConfigExecutionContext
	gvar   GlobalVariable
	mr     sync.RWMutex
	result []Result
}

func (r *initializer) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *initializer) initGvk(oc *OriginContext, v *ctrlcfgv1.GvkObject) *schema.GroupVersionKind {
	gvk, err := ctrlcfgv1.GetGVK(v.Resource)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	oc.GVK = gvk
	// initialize execution context for thr for and watch
	if oc.FOWS == FOWFor || oc.FOWS == FOWWatch {
		// initialize the gvk and rootVertex in the execution context
		if err := r.cec.Add(oc); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
	// initialize the output context
	r.gvar.Add(FOWEntry{FOW: oc.FOWS, RootVertexName: oc.VertexName})
	return gvk
}

func (r *initializer) initFunctionBlock(oc *OriginContext, v *ctrlcfgv1.FunctionElement) {
	if oc.BlockIndex >= 1 {
		// we can only have 1 block index -> only 1 recursion allowed
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("a pipeline van only have 1function block %v", *oc).Error(),
		})
	}
	if !v.Function.HasBlock() {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("a function block must have a block %v", *oc).Error(),
		})
	}
	if v.HasBlock() {
		if err := r.cec.AddBlock(oc); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
}
