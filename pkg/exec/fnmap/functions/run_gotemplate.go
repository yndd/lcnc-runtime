package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"text/template"

	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
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

func (r *gt) WithResult(result result.Result) {}

func (r *gt) WithNameAndNamespace(name, namespace string) {}

func (r *gt) WithClient(client client.Client) {}

func (r *gt) WithFnMap(fnMap fnmap.FuncMap) {}

func (r *gt) Run(ctx context.Context, vertexContext *rtdag.VertexContext, i input.Input) (output.Output, error) {
	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	r.outputs = vertexContext.Outputs
	if len(vertexContext.Function.Input.Resource.Raw) != 0 {
		r.template = string(vertexContext.Function.Input.Resource.Raw)
	} else {
		r.template = vertexContext.Function.Input.Template
	}

	// execute the function
	return r.fec.exec(ctx, vertexContext.Function, i)
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
	for varName, v := range r.outputs.Get() {
		oi, ok := v.(*output.OutputInfo)
		if !ok {
			return o, fmt.Errorf("expecting outputInfo, got %T", v)
		}
		o.AddEntry(varName, &output.OutputInfo{
			Internal: oi.Internal,
			GVK:      oi.GVK,
			Data:     r.output,
		})
	}
	return o, nil
}

func (r *gt) run(ctx context.Context, i input.Input) (any, error) {
	if r.template == "" {
		return nil, errors.New("missing template")
	}
	result := new(bytes.Buffer)
	// TODO: add template custom functions
	tpl, err := template.New("default").Option("missingkey=zero").Parse(r.template)
	if err != nil {
		return nil, err
	}
	fmt.Printf("runGT input: %v\n", i.Get())
	err = tpl.Execute(result, i.Get())
	if err != nil {
		return nil, err
	}
	var x any
	err = json.Unmarshal(result.Bytes(), &x)
	fmt.Printf("runGT result: %s", x)
	return x, err
}
