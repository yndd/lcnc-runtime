package fnmap

import (
	"context"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ForKey = "for"
)

func (r *fnmap) runForInit(ctx context.Context, req ctrl.Request, vertexContext *dag.VertexContext, input map[string]any) (map[string]*Output, error) {
	rx := &forQuery{
		outputContext: vertexContext.OutputContext,
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

	return fec.run(ctx, req, vertexContext.Function, input)
}

type forQuery struct {
	result        any
	outputContext map[string]*dag.OutputContext
}

func (r *forQuery) initResult(numItems int) {}

func (r *forQuery) recordResult(o any) { r.result = o }

func (r *forQuery) getResult() map[string]*Output {
	res := make(map[string]*Output, 1)
	for varName, outputCtx := range r.outputContext {
		res[varName] = &Output{
			Internal: outputCtx.Internal,
			Value:    r.result,
		}
	}
	return res
}

func (r *forQuery) prepareInput(fnconfig *ctrlcfgv1.Function) any { return fnconfig }

func (r *fnmap) forInit(ctx context.Context, req ctrl.Request, extraInput any, input map[string]any) (any, error) {
	for _, data := range input {
		return data, nil
	}
	return nil, nil
}
