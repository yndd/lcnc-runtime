package ccsyntax

import (
	"strings"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
)

func (r *parser) connect(d dag.DAG) []Result {
	c := &connector{
		d:      d,
		result: []Result{},
	}

	fnc := &WalkConfig{
		gvrObjectFn: c.connectGvr,
		functionFn:  c.connectFunction,
	}

	// walk the config resolve the verteces and create the outputmapping
	r.walkLcncConfig(fnc)
	// stop if errors were found
	return c.result
}

type connector struct {
	d              dag.DAG
	rootVertexName string
	mr             sync.RWMutex
	result         []Result
}

func (r *connector) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *connector) connectGvr(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvrObject) {
	r.rootVertexName = r.d.GetRootVertex()
}

func (r *connector) connectFunction(oc *OriginContext, v *ctrlcfgv1.ControllerConfigFunction) {
	if v.HasBlock() {
		r.connectBlock(oc, v.Block)
	}

	if v.Input != nil {
		if v.Input.Gvr != nil {
			r.d.Connect(r.rootVertexName, oc.VertexName)
		}
		if v.Input.Key != "" {
			r.connectRefs(oc, GetReferences(v.Input.Key))
		}
		if v.Input.Value != "" {
			r.connectRefs(oc, GetReferences(v.Input.Value))
		}
		for _, v := range v.Input.GenericInput {
			r.connectRefs(oc, GetReferences(v))
		}
	}
}

func (r *connector) connectBlock(oc *OriginContext, v *ctrlcfgv1.ControllerConfigBlock) {
	if v.Condition.Expression != nil {
		if v.Condition.Expression.Expression != nil {
			r.connectRefs(oc, GetReferences(*v.Condition.Expression.Expression))
			// continue to resolve if this is a nested block
			//r.connectBlock(oc, v.Condition.Block)
		}
	}
	if v.Range.Value != nil {
		if v.Range.Value != nil {
			r.connectRefs(oc, GetReferences(*v.Range.Value.Value))
			// continue to resolve if this is a nested block
			//r.connectBlock(oc, v.Range.Block)
		}
	}
}

func (r *connector) connectRefs(oc *OriginContext, refs []*Reference) {
	for _, ref := range refs {
		// RangeRefKind do nothing
		if ref.Kind == RegularReferenceKind {
			// get the vertexContext from the function
			vc := r.d.GetVertex(oc.VertexName)
			// lookup the localDAG first
			if vc.LocalVarDag != nil {
				if vc.LocalVarDag.Lookup(strings.Split(ref.Value, ".")) {
					// if the localVar lookup succeeds we are done -> continue
					continue
				}
			}
			vertexName, err := r.d.LookupRootVertex(strings.Split(ref.Value, "."))
			if err != nil {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         err.Error(),
				})
				continue
			}
			r.d.Connect(vertexName, oc.VertexName)
		}
	}
}
