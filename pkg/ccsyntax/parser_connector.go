package ccsyntax

import (
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) connect(ceCtx ConfigExecutionContext, outc OutputContext) []Result {
	c := &connector{
		ceCtx:  ceCtx,
		outc:   outc,
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
	ceCtx  ConfigExecutionContext
	outc   OutputContext
	mr     sync.RWMutex
	result []Result
}

func (r *connector) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *connector) connectGvk(oc *OriginContext, v *ctrlcfgv1.GvkObject) *schema.GroupVersionKind {
	gvk, err := ctrlcfgv1.GetGVK(v.Resource)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	return gvk
}

func (r *connector) connectFunction(oc *OriginContext, v *ctrlcfgv1.Function) {
	if v.HasVars() {
		oc := oc.DeepCopy()
		for localVarName, v := range v.Vars {
			oc.LocalVarName = localVarName
			r.connectRefs(oc, v)
			//r.connectFunction(oc, v)
		}
	}

	if v.HasBlock() {
		r.connectBlock(oc, v.Block)
	}

	if v.Input != nil {
		if len(v.Input.Resource.Raw) != 0 {
			r.ceCtx.GetDAGCtx(oc.FOW, oc.GVK, oc.Operation).DAG.Connect(oc.RootVertexName, oc.VertexName)
		}
		if v.Input.Key != "" {
			r.connectRefs(oc, v.Input.Key)
		}
		if v.Input.Value != "" {
			r.connectRefs(oc, v.Input.Value)
		}
		if v.Input.Expression != "" {
			r.connectRefs(oc, v.Input.Expression)
		}
		for _, v := range v.Input.GenericInput {
			r.connectRefs(oc, v)
		}
		if v.Input.Selector != nil {
			for k, v := range v.Input.Selector.MatchLabels {
				r.connectRefs(oc, k)
				r.connectRefs(oc, v)
			}
		}
	}

	// A block needs an explicit dependency to the root Block Vertex
	if oc.BlockIndex > 0 {
		r.connectVertex(oc, oc.BlockVertexName)
	}

	// A depndsOn needs an explicit dependency to the vertces they depend upon
	if len(v.DependsOn) != 0 {
		for _, vertexName := range v.DependsOn {
			r.connectVertex(oc, vertexName)
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
		// for regular values we resolve the variables
		// for varibales that start with _ this is a special case and
		// should only be used within a jq construct
		if ref.Kind == RegularReferenceKind && ref.Value[0] != '_' {
			// get the vertexContext from the function
			fmt.Printf("oc: %#v\n", oc)
			d := r.ceCtx.GetDAG(oc)
			vc := d.GetVertex(oc.VertexName)
			// lookup the localDAG first
			if vc.LocalVarDag != nil {
				if vc.LocalVarDag.VertexExists(ref.Value) {
					// the References are added to the runtime vertex context to ease processing
					vc.AddReference(ref.Value)
					// if the localVar lookup succeeds we are done -> continue
					continue
				}
			}
			// this is a global reference
			// the References are added to the runtime vertex context to ease processing
			vc.AddReference(ref.Value)
			vertexName, err := r.outc.GetDAG(FOWEntry{FOW: oc.FOW, RootVertexName: oc.RootVertexName}).GetOutputVertexName(ref.Value)
			if err != nil {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         err.Error(),
				})
				continue
			}
			r.ceCtx.GetDAGCtx(oc.FOW, oc.GVK, oc.Operation).DAG.Connect(vertexName, oc.VertexName)
		}
	}
}

func (r *connector) connectVertex(oc *OriginContext, vertexName string) {
	d := r.ceCtx.GetDAG(oc)
	d.Connect(vertexName, oc.VertexName)

}
