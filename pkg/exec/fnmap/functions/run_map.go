package functions

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/itchyny/gojq"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mapInput struct {
	key   string
	value string
}

type mapOutput struct {
	key   string
	value any
}

func NewMapFn() fnmap.Function {
	r := &kv{}

	r.fec = &fnExecConfig{
		executeRange:  true,
		executeSingle: false,
		// execution functions
		runFn: r.run,
		// result functions
		initOutputFn:     r.initOutput,
		recordOutputFn:   r.recordOutput,
		getFinalResultFn: r.getFinalResult,
	}

	return r
}

type kv struct {
	// fec exec config
	fec *fnExecConfig
	// init config
	// runtime config
	outputs output.Output
	key     string
	value   string
	// result, output
	m      sync.RWMutex
	output map[string]any
}

func (r *kv) Init(opts ...fnmap.FunctionOption) {
	for _, o := range opts {
		o(r)
	}
}

func (r *kv) WithOutput(output output.Output) {}

func (r *kv) WithResult(result result.Result) {}

func (r *kv) WithNameAndNamespace(name, namespace string) {}

func (r *kv) WithClient(client client.Client) {}

func (r *kv) WithFnMap(fnMap fnmap.FuncMap) {}

func (r *kv) Run(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (output.Output, error) {
	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	r.outputs = vertexContext.Outputs
	r.key = vertexContext.Function.Input.Key
	r.value = vertexContext.Function.Input.Value

	// execute the function
	return r.fec.exec(ctx, vertexContext.Function, input)
}

func (r *kv) initOutput(numItems int) {
	r.output = make(map[string]any, numItems)
}

func (r *kv) recordOutput(o any) {
	out, ok := o.(*mapOutput)
	if !ok {
		fmt.Printf("expecting mapOutput, got: %T", o)
	}
	r.m.Lock()
	defer r.m.Unlock()
	r.output[out.key] = out.value
}

func (r *kv) getFinalResult() (output.Output, error) {
	o := output.New()
	for varName, outputInfo := range r.outputs.GetOutputInfo() {
		o.RecordOutput(varName, &output.OutputInfo{
			Internal: outputInfo.Internal,
			GVK:      outputInfo.GVK,
			Value:    r.output,
		})
	}
	return o, nil
}

func (r *kv) run(ctx context.Context, input map[string]any) (any, error) {
	kv := &mapInput{
		key:   r.key,
		value: r.value,
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
		fmt.Printf("buildKV runJQOnce valC: %v\n", err.Error())
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
