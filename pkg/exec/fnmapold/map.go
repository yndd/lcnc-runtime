package fnmap

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/itchyny/gojq"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
)

type mapInput struct {
	key   string
	value string
}

type mapOutput struct {
	key   string
	value any
}

func (r *fnmap) runMap(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (map[string]*output.OutputInfo, error) {
	rx := &kv{
		outputContext: vertexContext.Outputs,
	}

	fec := &fnExecConfig{
		executeRange:  true,
		executeSingle: false,
		// execution functions
		prepareInputFn: rx.prepareInput,
		runFn:          rx.buildKV,
		// result functions
		initResultFn:   rx.initResult,
		recordResultFn: rx.recordResult,
		getResultFn:    rx.getResult,
	}

	return fec.run(ctx, vertexContext.Function, input)
}

type kv struct {
	m             sync.RWMutex
	result        map[string]any
	outputContext output.Output
}

func (r *kv) initResult(numItems int) {
	r.result = make(map[string]any, numItems)
}

func (r *kv) recordResult(o any) {
	out, ok := o.(*mapOutput)
	if !ok {
		fmt.Printf("expecting mapOutput, got: %T", o)
	}
	r.m.Lock()
	defer r.m.Unlock()
	r.result[out.key] = out.value
}

func (r *kv) getResult() map[string]*output.OutputInfo {
	res := make(map[string]*output.OutputInfo, 1)
	for varName, outputCtx := range r.outputContext.GetOutputInfo() {
		res[varName] = &output.OutputInfo{
			Internal: outputCtx.Internal,
			Value:    r.result,
		}
	}
	return res
}

func (r *kv) prepareInput(fnconfig *ctrlcfgv1.Function) any {
	return &mapInput{
		key:   fnconfig.Input.Key,
		value: fnconfig.Input.Value,
	}
}

func (r *kv) buildKV(ctx context.Context, extraInput any, input map[string]any) (any, error) {
	kv, ok := extraInput.(*mapInput)
	if !ok {
		return nil, fmt.Errorf("expecting mapInput with Key and value input, got: %T", extraInput)
	}
	if kv.value == "" {
		return nil, errors.New("missing input value")
	}
	if kv.key == "" {
		return nil, errors.New("missing input key")
	}
	varNames := make([]string, 0, len(input))
	varValues := make([]any, 0, len(input))
	for name, v := range input {
		varNames = append(varNames, "$"+name)
		varValues = append(varValues, v)
	}
	fmt.Printf("buildKV varNames: %v, varValues: %v\n", varNames, varValues)
	fmt.Printf("buildKV exp: %s\n", kv.value)

	valq, err := gojq.Parse(kv.value)
	if err != nil {
		fmt.Printf("buildKV valq: %s\n", err.Error())
		return nil, err
	}
	valC, err := gojq.Compile(valq, gojq.WithVariables(varNames))
	if err != nil {
		fmt.Printf("buildKV valC: %s\n", err.Error())
		return nil, err
	}
	keyq, err := gojq.Parse(kv.key)
	if err != nil {
		fmt.Printf("buildKV keyq: %s\n", err.Error())
		return nil, err
	}
	keyC, err := gojq.Compile(keyq, gojq.WithVariables(varNames))
	if err != nil {
		fmt.Printf("buildKV keyC: %s\n", err.Error())
		return nil, err
	}

	v, err := runJQOnce(valC, nil, varValues...)
	if err != nil {
		fmt.Printf("buildKV runJQOnce valC: %s\n", err.Error())
		return nil, err
	}

	k, err := runJQOnce(keyC, nil, varValues...)
	if err != nil {
		fmt.Printf("buildKV runJQOnce keyC: %s\n", err.Error())
		return nil, err
	}
	fmt.Printf("buildKV k: %T %#v\n", k, k)
	ks, ok := k.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected key format: %T", k)
	}
	return &mapOutput{key: ks, value: v}, nil
}

func runJQOnce(code *gojq.Code, input any, vars ...any) (any, error) {
	iter := code.Run(input, vars...)

	v, ok := iter.Next()
	if !ok {
		return nil, errors.New("no result")
	}
	if err, ok := v.(error); ok {
		if err != nil {
			fmt.Printf("runJQOnce err: %v\n", err)
			return nil, err
		}
	}
	return v, nil
}
