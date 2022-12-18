package ccsyntax

import (
	"strings"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) connect(ceCtx ConfigExecutionContext) []Result {
	c := &connector{
		ceCtx:  ceCtx,
		result: []Result{},
	}

	fnc := &WalkConfig{
		gvkObjectFn: c.connectGvk,
		functionFn:  c.connectFunction,
	}

	// walk the config resolve the verteces and create the outputmapping
	r.walkLcncConfig(fnc)
	// stop if errors were found
	return c.result
}

type connector struct {
	ceCtx          ConfigExecutionContext
	rootVertexName string
	mr             sync.RWMutex
	result         []Result
}

func (r *connector) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *connector) connectGvk(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvkObject) schema.GroupVersionKind {
	gvk, err := ctrlcfgv1.GetGVK(v.Resource)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	r.rootVertexName = r.ceCtx.GetDAG(oc.FOW, gvk).GetRootVertex()
	return gvk
}

func (r *connector) connectFunction(oc *OriginContext, v *ctrlcfgv1.ControllerConfigFunction) {
	if v.HasVars() {
		oc := oc.DeepCopy()
		for localVarName, v := range v.Vars {
			oc.LocalVarName = localVarName
			r.connectFunction(oc, v)
		}
	}

	if v.HasBlock() {
		r.connectBlock(oc, v.Block)
	}

	if v.Input.Selector != nil {
		for k, v := range v.Input.Selector.MatchLabels {
			r.connectRefs(oc, k)
			r.connectRefs(oc, v)
		}
	}

	if v.Input != nil {
		if len(v.Input.Resource.Raw) != 0 {
			r.ceCtx.GetDAG(oc.FOW, oc.GVK).Connect(r.rootVertexName, oc.VertexName)
		}
		if v.Input.Key != "" {
			r.connectRefs(oc, v.Input.Key)
		}
		if v.Input.Value != "" {
			r.connectRefs(oc, v.Input.Value)
		}
		for _, v := range v.Input.GenericInput {
			r.connectRefs(oc, v)
		}
	}
}

func (r *connector) connectBlock(oc *OriginContext, v ctrlcfgv1.Block) {
	if v.Range != nil {
		r.connectRefs(oc, v.Range.Value)
		// continue to resolve if this is a nested block
		if v.Range.Range != nil || v.Range.Condition != nil {
			r.connectBlock(oc, v.Range.Block)
		}
	}
	if v.Condition != nil {
		r.connectRefs(oc, v.Condition.Expression)
		// continue to resolve if this is a nested block
		if v.Condition.Range != nil || v.Condition.Condition != nil {
			r.connectBlock(oc, v.Condition.Block)
		}
	}
}

func (r *connector) connectRefs(oc *OriginContext, s string) {
	rfs := NewReferences()
	refs := rfs.GetReferences(s)

	for _, ref := range refs {
		// RangeRefKind do nothing
		if ref.Kind == RegularReferenceKind {
			// get the vertexContext from the function
			vc := r.ceCtx.GetDAG(oc.FOW, oc.GVK).GetVertex(oc.VertexName)
			// lookup the localDAG first
			if vc.LocalVarDag != nil {
				if vc.LocalVarDag.Lookup(strings.Split(ref.Value, ".")) {
					// if the localVar lookup succeeds we are done -> continue
					continue
				}
			}
			vertexName, err := r.ceCtx.GetDAG(oc.FOW, oc.GVK).LookupRootVertex(strings.Split(ref.Value, "."))
			if err != nil {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         err.Error(),
				})
				continue
			}
			r.ceCtx.GetDAG(oc.FOW, oc.GVK).Connect(vertexName, oc.VertexName)
		}
	}
}
