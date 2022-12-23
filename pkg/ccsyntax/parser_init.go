package ccsyntax

import (
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) init() (ConfigExecutionContext, []Result) {
	i := initializer{
		cec: NewConfigExecutionContext(r.cCfg.GetName()),
	}

	fnc := &WalkConfig{
		gvkObjectFn: i.initGvk,
	}
	// walk the config initialaizes the config execution context
	r.walkLcncConfig(fnc)

	return i.cec, i.result

}

type initializer struct {
	cec    ConfigExecutionContext
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
	// initialize execution context for thr for and watch
	if oc.FOW == FOWFor || oc.FOW == FOWWatch {
		// initialize the gvk in the execution context
		if err := r.cec.Add(oc.FOW, gvk, oc.VertexName); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
	return gvk
}
