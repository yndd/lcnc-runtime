package ccsyntax

import (
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/meta"
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

func (r *populator) addGvk(oc *OriginContext, v *ctrlcfgv1.GvkObject) *schema.GroupVersionKind {
	gvk, err := ctrlcfgv1.GetGVK(v.Resource)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	oc.GVK = gvk
	fmt.Printf("addGvk: gvk: %#v\n", *gvk)
	fmt.Printf("addGvk: oc: %#v\n", oc)
	if oc.FOW == FOWFor || oc.FOW == FOWWatch {
		if err := r.cec.GetDAG(oc.FOW, GVKOperation{GVK: *gvk, Operation: OperationApply}).AddVertex(oc.VertexName, &dag.VertexContext{
			Kind: dag.RootVertexKind,
			Function: &ctrlcfgv1.Function{
				Type: ctrlcfgv1.ForInitType,
				Input: &ctrlcfgv1.Input{
					Resource: v.Resource,
				},
			},
			OutputContext: map[string]*dag.OutputContext{
				oc.VertexName: {
					Internal: true,
					GVK:      gvk,
				},
			},
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
	// Own does not have a pipeline so there is no point in populating the DAG
	if oc.FOW == FOWFor {
		if err := r.cec.GetDAG(oc.FOW, GVKOperation{GVK: *gvk, Operation: OperationDelete}).AddVertex(oc.VertexName, &dag.VertexContext{
			Kind: dag.RootVertexKind,
			Function: &ctrlcfgv1.Function{
				Type: ctrlcfgv1.ForInitType,
				Input: &ctrlcfgv1.Input{
					Resource: v.Resource,
				},
			},
			OutputContext: map[string]*dag.OutputContext{
				oc.VertexName: {
					Internal: true,
					GVK:      gvk,
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

func (r *populator) addFunction(oc *OriginContext, v *ctrlcfgv1.Function) {
	// prepare the output context
	outputCtx := map[string]*dag.OutputContext{}
	gvkToVarName := map[string]string{}
	// add output in a seperate DAG
	outputDAG := dag.New()
	for varName, outputCfg := range v.Output {
		if err := outputDAG.AddVertex(varName, &dag.VertexContext{
			Kind: dag.OutputVertexKind,
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
		// prepare output context
		gvk, err := ctrlcfgv1.GetGVK(outputCfg.Resource)
		if err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
		outputCtx[varName] = &dag.OutputContext{
			Internal: outputCfg.Internal,
			GVK:      gvk,
		}
		gvkToVarName[meta.GVKToString(gvk)] = varName
	}
	// if no output, initialize the output Context variable with the vertexName
	if v.Output == nil {
		if v.Type == ctrlcfgv1.GoTemplate {
			if len(v.Input.Resource.Raw) != 0 {
				gvk, err := ctrlcfgv1.GetGVK(v.Input.Resource)
				if err != nil {
					r.recordResult(Result{
						OriginContext: oc,
						Error:         err.Error(),
					})
				}
				outputCtx[oc.VertexName] = &dag.OutputContext{
					Internal: false,
					GVK:      gvk,
				}
			}
			// TODO what to do for a template ??? How do i get a GVK, is it also an external resource
		} else {
			outputCtx[oc.VertexName] = &dag.OutputContext{
				Internal: true,
			}
		}
	}

	// add localVars in a seperate DAG
	// only used for resolution and dependencies
	localVarsDAG := dag.New()
	for localVarName := range v.Vars {
		if err := localVarsDAG.AddVertex(localVarName, &dag.VertexContext{
			Kind: dag.LocalVarVertexKind,
			//Function: v,
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}

	// add the function vertex to the dag
	if err := r.cec.GetDAG(oc.FOW, GVKOperation{GVK: *oc.GVK, Operation: oc.Operation}).AddVertex(oc.VertexName, &dag.VertexContext{
		Kind:          dag.FunctionVertexKind,
		OutputDAG:     outputDAG,
		LocalVarDag:   localVarsDAG,
		Function:      v,
		References:    []string{},   // initialize reference
		OutputContext: outputCtx,    // provide the preparsed output context to the vertex
		GVKToVerName:  gvkToVarName, // provide a preparsed mapping from gvk to varName
	}); err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
}
