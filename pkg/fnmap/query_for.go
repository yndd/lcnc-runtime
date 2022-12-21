package fnmap

import (
	"context"
	"fmt"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
)

const (
	ForKey = "for"
)

func (r *fnmap) runForQuery(ctx context.Context, req ctrl.Request, vertexContext *dag.VertexContext, input map[string]any) (map[string]*Output, error) {
	rx := &forQuery{
		outputContext: vertexContext.OutputContext,
	}

	fec := &fnExecConfig{
		executeRange:  false,
		executeSingle: true,
		// execution functions
		prepareInputFn: rx.prepareInput,
		runFn:          r.forQuery,
		// result functions
		initResultFn:   rx.initResult,
		recordResultFn: rx.recordResult,
		getResultFn:    rx.getResult,
	}

	return fec.run(ctx, req, vertexContext.Function, input)
}

type forQuery struct {
	result        any
	outputContext map[string]*dag.OutputContext
}

func (r *forQuery) initResult(numItems int) {}

func (r *forQuery) recordResult(o any) { r.result = o }

func (r *forQuery) getResult() map[string]*Output {
	res := make(map[string]*Output, 1)
	for varName, outputCtx := range r.outputContext {
		res[varName] = &Output{
			Internal: outputCtx.Internal,
			Value:    r.result,
		}
	}
	return res
}

func (r *forQuery) prepareInput(fnconfig *ctrlcfgv1.Function) any { return fnconfig }

func (r *fnmap) forQuery(ctx context.Context, req ctrl.Request, extraInput any, input map[string]any) (any, error) {
	// key is namespaced name
	key, ok := input[ForKey].(types.NamespacedName)
	if !ok {
		return nil, fmt.Errorf("unexpected type, expected namespacedName, got: %v", input[ForKey])
	}
	//o := getUnstructured(r.gvk)
	o := meta.GetUnstructuredFromGVK(r.gvk)
	if err := r.client.Get(ctx, key, o); err != nil {
		return nil, err
	}
	b, err := yaml.Marshal(o.UnstructuredContent())
	if err != nil {
		return nil, err
	}

	rj := map[string]interface{}{}
	if err := yaml.Unmarshal(b, &rj); err != nil {
		return nil, err
	}
	return rj, nil
}
