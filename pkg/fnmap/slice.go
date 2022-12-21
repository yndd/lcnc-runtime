package fnmap

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/itchyny/gojq"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	ctrl "sigs.k8s.io/controller-runtime"
	// ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

func (r *fnmap) runSlice(ctx context.Context, req ctrl.Request, vertexContext *dag.VertexContext, input map[string]any) (map[string]*Output, error) {
	rx := &slice{
		outputContext: vertexContext.OutputContext,
	}

	fec := &fnExecConfig{
		executeRange:  true,
		executeSingle: false,
		// execution functions
		prepareInputFn: rx.prepareInput,
		runFn:          rx.buildSliceItem,
		// result functions
		initResultFn:   rx.initResult,
		recordResultFn: rx.recordResult,
		getResultFn:    rx.getResult,
	}

	return fec.run(ctx, req, vertexContext.Function, input)
}

type slice struct {
	m             sync.RWMutex
	result        []any
	outputContext map[string]*dag.OutputContext
}

func (r *slice) initResult(numItems int) {
	r.result = make([]any, 0, numItems)
}

func (r *slice) recordResult(o any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.result = append(r.result, o)
}

func (r *slice) getResult() map[string]*Output {
	res := make(map[string]*Output, 1)
	for varName, outputCtx := range r.outputContext {
		res[varName] = &Output{
			Internal: outputCtx.Internal,
			Value:    r.result,
		}
	}
	return res
}

func (r *slice) prepareInput(fnconfig *ctrlcfgv1.Function) any {
	return fnconfig.Input.Value
}

func (r *slice) buildSliceItem(ctx context.Context, req ctrl.Request, extraInput any, input map[string]any) (any, error) {
	value, ok := extraInput.(string)
	if !ok {
		return nil, fmt.Errorf("expecting string input, got: %T", extraInput)
	}
	if value == "" {
		return nil, errors.New("missing input value")
	}
	varNames := make([]string, 0, len(input))
	varValues := make([]any, 0, len(input))
	for name, v := range input {
		varNames = append(varNames, "$"+name)
		varValues = append(varValues, v)
	}
	fmt.Printf("buildSliceItem varNames: %v, varValues: %v\n", varNames, varValues)
	fmt.Printf("buildSliceItem exp: %s\n", value)

	q, err := gojq.Parse(value)
	if err != nil {
		return nil, err
	}
	code, err := gojq.Compile(q, gojq.WithVariables(varNames))
	if err != nil {
		return nil, err
	}

	iter := code.Run(nil, varValues...)
	v, ok := iter.Next()
	if !ok {
		return nil, errors.New("no value")
	}
	if err, ok := v.(error); ok {
		if err != nil {
			fmt.Printf("buildSliceItem err: %v\n", err)
			return nil, err
		}
	}
	fmt.Printf("buildSliceItem value: %v\n", v)
	return v, nil
}
