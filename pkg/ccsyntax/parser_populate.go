package ccsyntax

import (
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
)

func (r *parser) populate(d dag.DAG) []Result {
	p := &populator{
		d:      d,
		result: []Result{},
	}

	fnc := &WalkConfig{
		gvkObjectFn: p.addGvr,
		functionFn:  p.addFunction,
	}

	// walk the config populate the verteces and create the hierarchical DAG
	// duplicate entries within a dag are checked
	r.walkLcncConfig(fnc)
	// stop if errors were found
	return p.result
}

type populator struct {
	d      dag.DAG
	mr     sync.RWMutex
	result []Result
}

func (r *populator) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *populator) addGvr(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvkObject) {
	if err := r.d.AddVertex(oc.VertexName, &dag.VertexContext{
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
	if err := r.d.AddVertex(oc.VertexName, &dag.VertexContext{
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
