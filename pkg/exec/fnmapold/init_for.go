package fnmap

import (
	"context"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
)

const (
	ForKey = "for"
)

func (r *fnmap) runForInit(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (map[string]*output.OutputInfo, error) {
	rx := &forQuery{
		outputContext: vertexContext.Outputs,
	}

	fec := &fnExecConfig{
		executeRange:  false,
		executeSingle: true,
		// execution functions
		prepareInputFn: rx.prepareInput,
		runFn:          r.forInit,
		// result functions
		initResultFn:   rx.initResult,
		recordResultFn: rx.recordResult,
		getResultFn:    rx.getResult,
	}

	return fec.run(ctx, vertexContext.Function, input)
}

type forQuery struct {
	result        any
	outputContext output.Output
}

func (r *forQuery) initResult(numItems int) {}

func (r *forQuery) recordResult(o any) { r.result = o }

func (r *forQuery) getResult() map[string]*output.OutputInfo {
	res := make(map[string]*output.OutputInfo, 1)
	for varName, outputCtx := range r.outputContext.GetOutputInfo() {
		res[varName] = &output.OutputInfo{
			Internal: outputCtx.Internal,
			Value:    r.result,
		}
	}
	return res
}

func (r *forQuery) prepareInput(fnconfig *ctrlcfgv1.Function) any { return fnconfig }

func (r *fnmap) forInit(ctx context.Context, extraInput any, input map[string]any) (any, error) {
	for _, data := range input {
		return data, nil
	}
	return nil, nil
}
