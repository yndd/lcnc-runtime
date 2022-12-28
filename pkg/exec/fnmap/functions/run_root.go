package functions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewRootFn() fnmap.Function {
	l := ctrl.Log.WithName("root fn")
	r := &root{
		l: l,
	}

	r.fec = &fnExecConfig{
		executeRange:  false,
		executeSingle: false,
		// execution functions
		// result functions
		getFinalResultFn: r.getFinalResult,
		l:                l,
	}
	return r
}

type root struct {
	// fec exec config
	fec *fnExecConfig
	// logging
	l logr.Logger
}

func (r *root) Init(opts ...fnmap.FunctionOption) {
	for _, o := range opts {
		o(r)
	}
}

func (r *root) WithOutput(output output.Output) {}

func (r *root) WithResult(result result.Result) {}

func (r *root) WithNameAndNamespace(name, namespace string) {}

func (r *root) WithClient(client client.Client) {}

func (r *root) WithFnMap(fnMap fnmap.FuncMap) {}

func (r *root) Run(ctx context.Context, vertexContext *rtdag.VertexContext, i input.Input) (output.Output, error) {
	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	// execute the function
	r.l.Info("run", "vertexName", vertexContext.VertexName, "input", i.Get())
	return r.fec.exec(ctx, vertexContext.Function, i)
}

func (r *root) getFinalResult() (output.Output, error) {
	return output.New(), nil
}
