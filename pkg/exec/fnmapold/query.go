package fnmap

import (
	"context"
	"fmt"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func (r *fnmap) runQuery(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (map[string]*output.OutputInfo, error) {
	rx := &query{
		outputContext: vertexContext.Outputs,
	}

	fec := &fnExecConfig{
		executeRange:  false,
		executeSingle: true,
		// execution functions
		prepareInputFn: rx.prepareInput,
		runFn:          r.query,
		// result functions
		initResultFn:   rx.initResult,
		recordResultFn: rx.recordResult,
		getResultFn:    rx.getResult,
	}

	return fec.run(ctx, vertexContext.Function, input)
}

type query struct {
	//m      sync.RWMutex
	result        any
	outputContext output.Output
}

func (r *query) initResult(numItems int) {}

func (r *query) recordResult(o any) { r.result = o }

func (r *query) getResult() map[string]*output.OutputInfo {
	res := make(map[string]*output.OutputInfo, 1)
	for varName, outputCtx := range r.outputContext.GetOutputInfo() {
		res[varName] = &output.OutputInfo{
			Internal: outputCtx.Internal,
			Value:    r.result,
		}
	}
	return res
}

func (r *query) prepareInput(fnconfig *ctrlcfgv1.Function) any { return fnconfig }

func (r *fnmap) query(ctx context.Context, extraInput any, input map[string]any) (any, error) {
	fnconfig, ok := extraInput.(*ctrlcfgv1.Function)
	if !ok {
		return nil, fmt.Errorf("expecting fnconfig input, got: %T", extraInput)
	}
	gvk, err := ctrlcfgv1.GetGVK(fnconfig.Input.Resource)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{}
	if fnconfig.Input.Selector != nil {
		// TODO namespace
		//opts = append(opts, client.InNamespace("x"))
		opts = append(opts, client.MatchingLabels(fnconfig.Input.Selector.MatchLabels))
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
