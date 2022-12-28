package functions

import (
	"context"
	"fmt"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func NewQueryFn() fnmap.Function {
	r := &query{}

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

type query struct {
	// fec exec config
	fec *fnExecConfig
	// init config
	client client.Client
	// runtime config
	outputs  output.Output
	resource runtime.RawExtension
	selector *metav1.LabelSelector
	// output, output
	output any
}

func (r *query) Init(opts ...fnmap.FunctionOption) {
	for _, o := range opts {
		o(r)
	}
}

func (r *query) WithOutput(output output.Output) {}

func (r *query) WithResult(result result.Result) {}

func (r *query) WithNameAndNamespace(name, namespace string) {}

func (r *query) WithClient(client client.Client) {
	r.client = client
}

func (r *query) WithFnMap(fnMap fnmap.FuncMap) {}

func (r *query) Run(ctx context.Context, vertexContext *rtdag.VertexContext, i input.Input) (output.Output, error) {
	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	r.outputs = vertexContext.Outputs
	r.resource = vertexContext.Function.Input.Resource
	//r.selector = vertexContext.Function.Input.Selector

	// execute to function
	return r.fec.exec(ctx, vertexContext.Function, i)
}

func (r *query) initOutput(numItems int) {}

func (r *query) recordOutput(o any) {
	r.output = o
}

func (r *query) getFinalResult() (output.Output, error) {
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
	o.Print()
	return o, nil
}

func (r *query) run(ctx context.Context, i input.Input) (any, error) {
	gvk, err := ctrlcfgv1.GetGVK(r.resource)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Query: gvk: %v\n", *gvk)

	opts := []client.ListOption{}
	if r.selector != nil {
		// TODO namespace
		//opts = append(opts, client.InNamespace("x"))
		opts = append(opts, client.MatchingLabels(r.selector.MatchLabels))
	}

	o := meta.GetUnstructuredListFromGVK(gvk)
	if err := r.client.List(ctx, o, opts...); err != nil {
		return nil, err
	}

	rj := make([]interface{}, 0, len(o.Items))
	for _, v := range o.Items {
		b, err := yaml.Marshal(v.UnstructuredContent())
		if err != nil {
			return nil, err
		}

		vrj := map[string]interface{}{}
		if err := yaml.Unmarshal(b, &vrj); err != nil {
			return nil, err
		}
		rj = append(rj, vrj)
	}

	return rj, nil
}
