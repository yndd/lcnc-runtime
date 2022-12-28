package ccsyntax

import (
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/ccsyntax/vardag"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) populate(cec ConfigExecutionContext, gvar GlobalVariable) []Result {
	p := &populator{
		cec:    cec,
		gvar:   gvar,
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
	gvar   GlobalVariable
	mr     sync.RWMutex
	result []Result
}

func (r *populator) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *populator) addGvk(oc *OriginContext, v *ctrlcfgv1.GvkObject) *schema.GroupVersionKind {
	// a gvk is needed for each rootVertex
	gvk, err := ctrlcfgv1.GetGVK(v.Resource)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	oc.GVK = gvk

	outputs := output.New()
	outputs.AddEntry(oc.VertexName, &output.OutputInfo{
		Internal: true,
		GVK:      gvk,
	})

	// add the runtime outputCtxt to the outputCtxt DAG for ensuring the output varibales are globally unique
	// and to resolve and connect the graph
	r.gvar.Add(FOWEntry{FOW: oc.FOW, RootVertexName: oc.RootVertexName})
	if err := r.gvar.GetDAG(FOWEntry{FOW: oc.FOW, RootVertexName: oc.RootVertexName}).AddVariable(oc.VertexName, &vardag.VariableContext{
		OutputVertex:    oc.VertexName,
		BlockIndex:      oc.BlockIndex,
		BlockVertexName: oc.BlockVertexName,
	}); err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}

	// add the vertexContext to the runtime DAG
	// OWN does not have a runtime DAG
	// FOR has both an apply and delete runtime DAG
	// WATCH has only an apply runtime DAG
	if oc.FOW == FOWFor || oc.FOW == FOWWatch {
		if err := r.cec.GetDAGCtx(oc.FOW, oc.GVK, OperationApply).DAG.AddVertex(oc.VertexName, &rtdag.VertexContext{
			VertexName: oc.VertexName,
			Kind:       rtdag.RootVertexKind,
			Function: &ctrlcfgv1.Function{
				Type: ctrlcfgv1.RootType,
				Input: &ctrlcfgv1.Input{
					Resource: v.Resource,
				},
			},
			Outputs: outputs,
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
	if oc.FOW == FOWFor {
		if err := r.cec.GetDAGCtx(oc.FOW, oc.GVK, OperationDelete).DAG.AddVertex(oc.VertexName, &rtdag.VertexContext{
			VertexName: oc.VertexName,
			Kind:       rtdag.RootVertexKind,
			Function: &ctrlcfgv1.Function{
				Type: ctrlcfgv1.RootType,
				Input: &ctrlcfgv1.Input{
					Resource: v.Resource,
				},
			},
			Outputs: outputs,
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

	// prepare the output context such that the runtime processing is easier
	outputs := output.New()
	gvkToVarName := map[string]string{}
	for varName, outputCfg := range v.Output {
		// prepare output context
		gvk, err := ctrlcfgv1.GetGVK(outputCfg.Resource)
		if err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
		outputs.AddEntry(varName, &output.OutputInfo{
			Internal: outputCfg.Internal,
			GVK:      gvk,
		})
		gvkToVarName[meta.GVKToString(gvk)] = varName

		// add the runtime outputCtxt to the outputCtxt DAG for ensuring the output varibales are globally unique
		// and to resolve and connect the graph
		if err := r.gvar.GetDAG(FOWEntry{FOW: oc.FOW, RootVertexName: oc.RootVertexName}).AddVariable(varName, &vardag.VariableContext{
			OutputVertex:    oc.VertexName,
			BlockIndex:      oc.BlockIndex,
			BlockVertexName: oc.BlockVertexName,
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
	// if no output, initialize the output Context variable with the vertexName
	if v.Output == nil {
		if v.Type == ctrlcfgv1.GoTemplateType {
			if len(v.Input.Resource.Raw) != 0 {
				gvk, err := ctrlcfgv1.GetGVK(v.Input.Resource)
				if err != nil {
					r.recordResult(Result{
						OriginContext: oc,
						Error:         err.Error(),
					})
				}
				outputs.AddEntry(oc.VertexName, &output.OutputInfo{
					Internal: false,
					GVK:      gvk,
				})
			}
			// TODO what to do for a template ??? How do i get a GVK, is it also an external resource
		} else {
			if v.Input != nil && len(v.Input.Resource.Raw) != 0 {
				gvk, err := ctrlcfgv1.GetGVK(v.Input.Resource)
				if err != nil {
					r.recordResult(Result{
						OriginContext: oc,
						Error:         err.Error(),
					})
				}
				outputs.AddEntry(oc.VertexName, &output.OutputInfo{
					Internal: true,
					GVK:      gvk,
				})
			} else {
				outputs.AddEntry(oc.VertexName, &output.OutputInfo{
					Internal: true,
				})
			}
		}
		// add the runtime outputCtxt to the outputCtxt DAG for ensuring the output varibales are globally unique
		// and to resolve and connect the graph
		if err := r.gvar.GetDAG(FOWEntry{FOW: oc.FOW, RootVertexName: oc.RootVertexName}).AddVariable(oc.VertexName, &vardag.VariableContext{
			OutputVertex:    oc.VertexName,
			BlockIndex:      oc.BlockIndex,
			BlockVertexName: oc.BlockVertexName,
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}

	// add the function vertex to the dag
	// if there is a functionblock we could have a different DAG -> select the right dag
	// BlockDAG = nil -> no range -> process regularly and add everything to the main runtimeDAG
	// BlockDAG NOT nil
	// + oc.BlockIndex == 0 this is the initial block index and we need to add the vertex to both the main runtimeDAG
	//   and the block runtime DAG -> in the main runtimeDAG add the blockDAG
	// + oc.BlockIndex !=0 (1) this is a block DAG, so process regurlary in the block DAG
	if !oc.Block {
		// this is a regular entry in the main runtime DAG
		if err := r.cec.GetDAG(oc).AddVertex(oc.VertexName, &rtdag.VertexContext{
			VertexName:   oc.VertexName,
			Kind:         rtdag.FunctionVertexKind,
			Function:     v,
			References:   []string{},   // initialize reference
			Outputs:      outputs,      // provide the preparsed output context to the vertex
			GVKToVarName: gvkToVarName, // provide a preparsed mapping from gvk to varName
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
		return
	}
	// This is a block
	rootDAG := r.cec.GetDAG(oc)
	if oc.BlockVertexName == "" {
		oc.BlockVertexName = oc.VertexName
	}
	blockDAG := r.cec.GetDAG(oc)
	if oc.BlockIndex == 0 {
		// this is the initial block index and we need to add the vertex to both the main runtimeDAG
		// and the block runtime DAG -> in the main runtimeDAG add the blockDAG
		if err := rootDAG.AddVertex(oc.VertexName, &rtdag.VertexContext{
			VertexName:   oc.VertexName,
			Kind:         rtdag.FunctionVertexKind,
			BlockDAG:     blockDAG,
			Function:     v,
			References:   []string{},   // initialize reference
			Outputs:      outputs,      // provide the preparsed output context to the vertex
			GVKToVarName: gvkToVarName, // provide a preparsed mapping from gvk to varName
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
		// add the vertex to the blockDAG
		if err := blockDAG.AddVertex(oc.VertexName, &rtdag.VertexContext{
			VertexName: oc.VertexName,
			Kind:       rtdag.RootVertexKind, // this is the rootVertex in the blockDAG
			Function: &ctrlcfgv1.Function{
				Type: ctrlcfgv1.RootType,
			},
			References:   []string{},   // initialize reference
			Outputs:      outputs,      // provide the preparsed output context to the vertex
			GVKToVarName: gvkToVarName, // provide a preparsed mapping from gvk to varName
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	} else {
		if err := blockDAG.AddVertex(oc.VertexName, &rtdag.VertexContext{
			VertexName:   oc.VertexName,
			Kind:         rtdag.FunctionVertexKind,
			Function:     v,
			References:   []string{},   // initialize reference
			Outputs:      outputs,      // provide the preparsed output context to the vertex
			GVKToVarName: gvkToVarName, // provide a preparsed mapping from gvk to varName
		}); err != nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         err.Error(),
			})
		}
	}
}
