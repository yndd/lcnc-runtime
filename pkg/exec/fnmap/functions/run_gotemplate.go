package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"text/template"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewGTFn() fnmap.Function {
	r := &gt{}

	r.fec = &fnExecConfig{
		executeRange:  true,
		executeSingle: true,
		// execution functions
		runFn: r.run,
		// result functions
		initOutputFn:     r.initOutput,
		recordOutputFn:   r.recordOutput,
		getFinalResultFn: r.getFinalResult,
	}
	return r
}

type gt struct {
	// fec exec config
	fec *fnExecConfig
	// init config
	// runtime config
	outputs  output.Output
	template string
	// result, output
	m      sync.RWMutex
	output []any
}

func (r *gt) Init(opts ...fnmap.FunctionOption) {
	for _, o := range opts {
		o(r)
	}
}

func (r *gt) WithOutput(output output.Output) {}

func (r *gt) WithNameAndNamespace(name, namespace string) {}

func (r *gt) WithClient(client client.Client) {}

func (r *gt) WithFnMap(fnMap fnmap.FuncMap) {}

func (r *gt) Run(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (output.Output, error) {
	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	r.outputs = vertexContext.Outputs
	if len(vertexContext.Function.Input.Resource.Raw) != 0 {
		r.template = string(vertexContext.Function.Input.Resource.Raw)
	} else {
		r.template = vertexContext.Function.Input.Template
	}

	// execute the function
	return r.fec.exec(ctx, vertexContext.Function, input)
}

func (r *gt) initOutput(numItems int) {
	r.output = make([]any, 0, numItems)
}

func (r *gt) recordOutput(o any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.output = append(r.output, o)
}

func (r *gt) getFinalResult() (output.Output, error) {
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

func (r *gt) run(ctx context.Context, input map[string]any) (any, error) {
	if r.template == "" {
		return nil, errors.New("missing template")
	}
	result := new(bytes.Buffer)
	// TODO: add template custom functions
	tpl, err := template.New("default").Option("missingkey=zero").Parse(r.template)
	if err != nil {
		return nil, err
	}
	fmt.Printf("runGT input: %v\n", input)
	err = tpl.Execute(result, input)
	if err != nil {
		return nil, err
	}
	var x any
	err = json.Unmarshal(result.Bytes(), &x)
	fmt.Printf("runGT result: %s", x)
	return x, err
}
