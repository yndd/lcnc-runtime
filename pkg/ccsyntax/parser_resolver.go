package ccsyntax

import (
	"fmt"
	"strings"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
)

func (r *parser) resolve(d dag.DAG) []Result {
	rs := &resolver{
		d:      d,
		result: []Result{},
	}

	fnc := &WalkConfig{
		gvrObjectFn: rs.resolveGvr,
		functionFn:  rs.resolveFunction,
	}

	// walk the config resolve the verteces and create the outputmapping
	r.walkLcncConfig(fnc)
	// stop if errors were found
	return rs.result
}

type resolver struct {
	d      dag.DAG
	mr     sync.RWMutex
	result []Result
}

func (r *resolver) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *resolver) resolveGvr(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvrObject) {
	// nothing todo
}

func (r *resolver) resolveFunction(oc *OriginContext, v *ctrlcfgv1.ControllerConfigFunction) {
	if v.HasVars() {
		oc := oc.DeepCopy()
		for localVarName, v := range v.Vars {
			oc.LocalVarName = localVarName
			r.resolveFunction(oc, v)
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
	for _, v := range v.Input.GenericInput {
		r.resolveRefs(oc, v)
	}
}

func (r *resolver) resolveBlock(oc *OriginContext, v *ctrlcfgv1.Block) {
	if v.Range != nil {
		r.resolveRefs(oc, v.Range.Value)
		// continue to resolve if this is a nested block
		if v.Range.Block != nil {
			r.resolveBlock(oc, v.Range.Block)
		}

	}
	if v.Condition != nil {
		r.resolveRefs(oc, v.Condition.Expression)
		// continue to resolve if this is a nested block
		if v.Condition.Block != nil {
			r.resolveBlock(oc, v.Condition.Block)
		}
	}

}

func (r *resolver) resolveRefs(oc *OriginContext, s string) {
	rfs := NewReferences()
	refs := rfs.GetReferences(s)

	for _, ref := range refs {
		// for val
		if ref.Kind == RegularReferenceKind {
			// get the vertexContext from the function
			vc := r.d.GetVertex(oc.VertexName)
			// lookup the localDAG first
			if vc.LocalVarDag != nil {
				if vc.LocalVarDag.Lookup(strings.Split(ref.Value, ".")) {
					// if the lookup succeeds we are done
					continue
				}
			}
			// if the lookup in the root DAG does not succeed we record the result
			// and fail eventually
			if !r.d.Lookup(strings.Split(ref.Value, ".")) {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         fmt.Errorf("cannot resolve %s", ref.Value).Error(),
				})
			}
		}
	}
}
