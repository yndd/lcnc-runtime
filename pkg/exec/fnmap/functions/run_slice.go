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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewSliceFn() fnmap.Function {
	r := &slice{}

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

type slice struct {
	// fec exec config
	fec *fnExecConfig
	// init config
	// runtime config
	outputs output.Output
	value   string
	// result, output
	m      sync.RWMutex
	output []any
}

func (r *slice) Init(opts ...fnmap.FunctionOption) {
	for _, o := range opts {
		o(r)
	}
}

func (r *slice) WithOutput(output output.Output) {}

func (r *slice) WithNameAndNamespace(name, namespace string) {}

func (r *slice) WithClient(client client.Client) {}

func (r *slice) WithFnMap(fnMap fnmap.FuncMap) {}

func (r *slice) Run(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (output.Output, error) {
	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	r.outputs = vertexContext.Outputs
	r.value = vertexContext.Function.Input.Value

	// execute the function
	return r.fec.exec(ctx, vertexContext.Function, input)
}

func (r *slice) initOutput(numItems int) {
	r.output = make([]any, 0, numItems)
}

func (r *slice) recordOutput(o any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.output = append(r.output, o)
}

func (r *slice) getFinalResult() (output.Output, error) {
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

func (r *slice) run(ctx context.Context, input map[string]any) (any, error) {
	if r.value == "" {
		return nil, errors.New("missing input value")
	}
	varNames := make([]string, 0, len(input))
	varValues := make([]any, 0, len(input))
	for name, v := range input {
		varNames = append(varNames, "$"+name)
		varValues = append(varValues, v)
	}
	fmt.Printf("buildSliceItem varNames: %v, varValues: %v\n", varNames, varValues)
	fmt.Printf("buildSliceItem exp: %s\n", r.value)

	q, err := gojq.Parse(r.value)
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
