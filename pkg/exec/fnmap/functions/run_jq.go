package functions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewJQFn() fnmap.Function {
	l := ctrl.Log.WithName("jq fn")
	r := &jq{
		l: l,
	}

	r.fec = &fnExecConfig{
		executeRange:  false,
		executeSingle: true,
		// execution functions
		filterInputFn: r.filterInput,
		runFn: r.run,
		// result functions
		initOutputFn:     r.initOutput,
		recordOutputFn:   r.recordOutput,
		getFinalResultFn: r.getFinalResult,
		l:                l,
	}
	return r
}

type jq struct {
	// fec exec config
	fec *fnExecConfig
	// init config
	// runtime config
	outputs    output.Output
	expression string
	// result, output
	output any
	// logging
	l logr.Logger
}

func (r *jq) Init(opts ...fnmap.FunctionOption) {
	for _, o := range opts {
		o(r)
	}
}

func (r *jq) WithOutput(output output.Output) {}

func (r *jq) WithResult(result result.Result) {}

func (r *jq) WithNameAndNamespace(name, namespace string) {}

func (r *jq) WithRootVertexName(name string) {}

func (r *jq) WithClient(client client.Client) {}

func (r *jq) WithFnMap(fnMap fnmap.FuncMap) {}

func (r *jq) Run(ctx context.Context, vertexContext *rtdag.VertexContext, i input.Input) (output.Output, error) {
	r.l.Info("run", "vertexName", vertexContext.VertexName, "input", i.Get(), "expression", vertexContext.Function.Input.Expression)

	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	r.outputs = vertexContext.Outputs
	r.expression = vertexContext.Function.Input.Expression
	// execute the function
	return r.fec.exec(ctx, vertexContext.Function, i)
}

func (r *jq) initOutput(numItems int) {}

func (r *jq) recordOutput(o any) {
	r.output = o
}

func (r *jq) getFinalResult() (output.Output, error) {
	o := output.New()
	for varName, v := range r.outputs.Get() {
		//fmt.Printf("query getFinalResult varName: %s, outputInfo %#v\n", varName, *outputInfo)
		//fmt.Printf("query getFinalResult value: %#v\n", r.output)
		oi, ok := v.(*output.OutputInfo)
		if !ok {
			err := fmt.Errorf("expecting outputInfo, got %T", v)
			r.l.Error(err, "cannot record result")
			return o, err
		}
		o.AddEntry(varName, &output.OutputInfo{
			Internal: oi.Internal,
			GVK:      oi.GVK,
			Data:     r.output,
		})
	}
	return o, nil
}

func (r *jq) filterInput(i input.Input) input.Input {return i}

func (r *jq) run(ctx context.Context, i input.Input) (any, error) {
	return runJQ(r.expression, i)
}
