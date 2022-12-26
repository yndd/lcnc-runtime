package functions

import (
	"context"
	"fmt"
	"sync"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/executor"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewBlockFn() fnmap.Function {
	r := &block{}

	r.fec = &fnExecConfig{
		executeRange:  false,
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

type block struct {
	// fec exec config
	fec *fnExecConfig
	// init config
	curOutputs output.Output // this is the current output list
	curResults result.Result
	fnMap      fnmap.FuncMap
	// runtime config
	d dag.DAG
	// result, output
	m      sync.RWMutex
	output []any
}

func (r *block) Init(opts ...fnmap.FunctionOption) {
	for _, o := range opts {
		o(r)
	}
}

func (r *block) WithOutput(output output.Output) {
	r.curOutputs = output
}

func (r *block) WithResult(result result.Result) {
	r.curResults = result
}

func (r *block) WithNameAndNamespace(name, namespace string) {}

func (r *block) WithClient(client client.Client) {}

func (r *block) WithFnMap(fnMap fnmap.FuncMap) {
	r.fnMap = fnMap
}

func (r *block) Run(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (output.Output, error) {
	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	r.d = vertexContext.BlockDAG

	// execute to function
	return r.fec.exec(ctx, vertexContext.Function, input)
}

func (r *block) initOutput(numItems int) {
	r.output = make([]any, 0, numItems)
}

func (r *block) recordOutput(o any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.output = append(r.output, o)
}

func (r *block) getFinalResult() (output.Output, error) {
	if len(r.output) == 0 {
		return output.New(), nil
	}
	return output.New(), nil
}

func (r *block) run(ctx context.Context, input map[string]any) (any, error) {
	// check if the dag is initialized
	if r.d == nil {
		return nil, fmt.Errorf("expecting an initialized dag, got: %T", r.d)
	}

	// debug
	r.d.PrintVertices()
	fmt.Printf("block root Vertex: %s\n", r.d.GetRootVertex())

	e := executor.New(&executor.Config{
		Type:           result.ExecBlockType,
		Name:           r.d.GetRootVertex(),
		RootVertexName: r.d.GetRootVertex(),
		DAG:            r.d,
		FnMap:          r.fnMap,
		Output:         r.curOutputs,
		Result:         r.curResults,
	})
	e.Run(ctx)

	return nil, nil

}
