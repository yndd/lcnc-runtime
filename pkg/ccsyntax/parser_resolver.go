package ccsyntax

import (
	"fmt"
	"strings"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

func (r *parser) resolve(ceCtx ConfigExecutionContext) []Result {
	rs := &resolver{
		ceCtx:  ceCtx,
		result: []Result{},
	}

	fnc := &WalkConfig{
		//gvkObjectFn: rs.resolveGvk,
		functionFn: rs.resolveFunction,
	}

	// walk the config resolve the verteces and create the outputmapping
	r.walkLcncConfig(fnc)
	// stop if errors were found
	return rs.result
}

type resolver struct {
	ceCtx  ConfigExecutionContext
	mr     sync.RWMutex
	result []Result
}

func (r *resolver) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *resolver) resolveFunction(oc *OriginContext, v *ctrlcfgv1.Function) {
	if v.HasVars() {
		oc := oc.DeepCopy()
		for localVarName, v := range v.Vars {
			oc.LocalVarName = localVarName
			r.resolveRefs(oc, v)
		}
	}

	if v.HasBlock() {
		r.resolveBlock(oc, v.Block)
	}

	if v.Input.Selector != nil {
		for k, v := range v.Input.Selector.MatchLabels {
			r.resolveRefs(oc, k)
			r.resolveRefs(oc, v)
		}
	}

	if v.Input.Key != "" {
		r.resolveRefs(oc, v.Input.Key)
	}
	if v.Input.Value != "" {
		r.resolveRefs(oc, v.Input.Value)
	}
	if v.Input.Expression != "" {
		r.resolveRefs(oc, v.Input.Expression)
	}
	for _, v := range v.Input.GenericInput {
		r.resolveRefs(oc, v)
	}
	if len(v.DependsOn) > 0 {
		r.resolveDependsOn(oc, v.DependsOn)
	}
}

func (r *resolver) resolveBlock(oc *OriginContext, v ctrlcfgv1.Block) {
	if v.Range != nil {
		r.resolveRefs(oc, v.Range.Value)
		// continue to resolve if this is a nested block
		if v.Range.Range != nil || v.Range.Condition != nil {
			r.resolveBlock(oc, v.Range.Block)
		}

	}
	if v.Condition != nil {
		r.resolveRefs(oc, v.Condition.Expression)
		// continue to resolve if this is a nested block
		if v.Condition.Range != nil || v.Condition.Condition != nil {
			r.resolveBlock(oc, v.Condition.Block)
		}
	}

}

func (r *resolver) resolveRefs(oc *OriginContext, s string) {
	rfs := NewReferences()
	refs := rfs.GetReferences(s)

	for _, ref := range refs {
		// for regular values we resolve the variables
		// for varibales that start with _ this is a special case and
		// should only be used within a jq construct
		if ref.Kind == RegularReferenceKind && ref.Value[0] != '_' {
			// get the vertexContext from the function
			vc := r.ceCtx.GetDAG(oc.FOW, GVKOperation{GVK: *oc.GVK, Operation: oc.Operation}).GetVertex(oc.VertexName)
			// lookup the localDAG first
			if vc.LocalVarDag != nil {
				if vc.LocalVarDag.Lookup(strings.Split(ref.Value, ".")) {
					// if the lookup succeeds we are done
					continue
				}
			}
			// if the lookup in the root DAG does not succeed we record the result
			// and fail eventually
			if !r.ceCtx.GetDAG(oc.FOW, GVKOperation{GVK: *oc.GVK, Operation: oc.Operation}).Lookup(strings.Split(ref.Value, ".")) {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         fmt.Errorf("cannot resolve %s", ref.Value).Error(),
				})
			}
		}
	}
}

func (r *resolver) resolveDependsOn(oc *OriginContext, vertexNames []string) {
	for _, vertexName := range vertexNames {
		if r.ceCtx.GetDAG(oc.FOW, GVKOperation{GVK: *oc.GVK, Operation: oc.Operation}).GetVertex(vertexName) == nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("vertex in depndsOn does not exist %s", vertexName).Error(),
			})
		}
	}
}
