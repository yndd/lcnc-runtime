package fnmap

import (
	"context"
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
)

func (r *fnmap) runBlock(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (map[string]*output.OutputInfo, error) {
	rx := &block{
		rootVertexName: r.rootVertexName,
		DAG:            vertexContext.BlockDAG,
		output:         r.output,
		fnMap:          r,
	}

	fec := &fnExecConfig{
		executeRange:  true,
		executeSingle: true,
		// execution functions
		prepareInputFn: rx.prepareInput,
		runFn:          rx.runBlockDAG,
		// result functions
		initResultFn:   rx.initResult,
		recordResultFn: rx.recordResult,
		getResultFn:    rx.getResult,
	}

	return fec.run(ctx, vertexContext.Function, input)
}

type block struct {
	m      sync.RWMutex
	result []any
	// outputContext map[string]*dag.OutputContext
	rootVertexName string
	DAG            dag.DAG
	output         output.Output
	fnMap          FnMap
}

func (r *block) initResult(numItems int) {
	r.result = make([]any, 0, numItems)
}

func (r *block) recordResult(o any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.result = append(r.result, o)
}

func (r *block) getResult() map[string]*output.OutputInfo {
	// this is typically used for a condition == true
	if len(r.result) == 0 {
		return map[string]*output.OutputInfo{}
	}
	// TODO Range
	return map[string]*output.OutputInfo{}
}

func (r *block) prepareInput(fnconfig *ctrlcfgv1.Function) any {
	return nil
}

func (r *block) runBlockDAG(ctx context.Context, extraInput any, input map[string]any) (any, error) {
	// TODO for a condition we will not come here
	// For a Range we have to execute the dag + execution context
	if r.DAG == nil {
		return nil, fmt.Errorf("expecting an initialized dag, got: %T", r.DAG)
	}

	// TODO
	// How to get access to the global output

	//e := executor.New(&executor.Config{})
	/*
		e := executor.New(&executor.Config{
			RootVertexName: r.rootVertexName,
			DAG:            r.DAG,
			FnMap:          r.fnMap,
			Output:         r.output,
		})

		result := e.Run(ctx)
		return result, nil
	*/
	return nil, nil

}
