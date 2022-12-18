package ccsyntax

import (
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) populate(cec ConfigExecutionContext) []Result {
	p := &populator{
		cec:    cec,
		result: []Result{},
	}

	fnc := &WalkConfig{
		gvkObjectFn: p.addGvk,
		functionFn:  p.addFunction,
	}

	// walk the config populate the verteces and create the hierarchical DAG
	// duplicate entries within a dag are checked
	r.walkLcncConfig(fnc)
	// stop if errors were found
	return p.result
}

type populator struct {
	cec    ConfigExecutionContext
	mr     sync.RWMutex
	result []Result
}

func (r *populator) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *populator) addGvk(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvkObject) schema.GroupVersionKind {
	gvk, err := ctrlcfgv1.GetGVK(v.Resource)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	// Own does not have a pipeline so there is no point in populating the DAG
	if oc.FOW != FOWOwn && err == nil {
		if err := r.cec.GetDAG(oc.FOW, gvk).AddVertex(oc.VertexName, &dag.VertexContext{
			Kind: dag.RootVertexKind,
			Function: &ctrlcfgv1.ControllerConfigFunction{
				Type: ctrlcfgv1.ForQueryType,
				Input: &ctrlcfgv1.ControllerConfigInput{
					Resource: v.Resource,
				},
			},
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
	return gvk
}

func (r *populator) addFunction(oc *OriginContext, v *ctrlcfgv1.ControllerConfigFunction) {
	// add output in a seperate DAG
	outputDAG := dag.New()
	for outputName := range v.Output {
		if err := outputDAG.AddVertex(outputName, &dag.VertexContext{
			Kind: dag.OutputVertexKind,
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}

	// add localVars in a seperate DAG
	// only used for resolution and dependencies
	localVarsDAG := dag.New()
	for localVarName, v := range v.Vars {
		if err := localVarsDAG.AddVertex(localVarName, &dag.VertexContext{
			Kind:     dag.LocalVarVertexKind,
			Function: v,
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}

	// add the function vertex to the dag
	if err := r.cec.GetDAG(oc.FOW, oc.GVK).AddVertex(oc.VertexName, &dag.VertexContext{
		Kind:        dag.FunctionVertexKind,
		OutputDAG:   outputDAG,
		LocalVarDag: localVarsDAG,
		Function:    v,
	}); err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
}
