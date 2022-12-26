package functions

import (
	"context"
	"fmt"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewRootFn() fnmap.Function {
	r := &root{}

	r.fec = &fnExecConfig{
		executeRange:  false,
		executeSingle: false,
		// execution functions
		// result functions
		getFinalResultFn: r.getFinalResult,
	}
	return r
}

type root struct {
	// fec exec config
	fec *fnExecConfig
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

func (r *root) Run(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (output.Output, error) {
	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	fmt.Printf("run root kind: %s\n", vertexContext.Name)
	// execute the function
	return r.fec.exec(ctx, vertexContext.Function, input)
}

func (r *root) getFinalResult() (output.Output, error) {
	return output.New(), nil
}
