package fnmap

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	rctxv1 "github.com/yndd/lcnc-runtime/pkg/api/resourcecontext/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/fnruntime"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *fnmap) runImage(ctx context.Context, req ctrl.Request, vertexContext *dag.VertexContext, input map[string]any) (map[string]*Output, error) {
	rx := &image{
		outputContext: vertexContext.OutputContext,
		gvkToVarName:  vertexContext.GVKToVerName,
	}

	fec := &fnExecConfig{
		executeRange:  true,
		executeSingle: true,
		// execution functions
		prepareInputFn: rx.prepareInput,
		runFn:          rx.runImage,
		// result functions
		initResultFn:   rx.initResult,
		recordResultFn: rx.recordResult,
		getResultFn:    rx.getResult,
	}

	return fec.run(ctx, req, vertexContext.Function, input)
}

type image struct {
	m             sync.RWMutex
	result        map[string]*Output
	numItems      int
	outputContext map[string]*dag.OutputContext
	gvkToVarName  map[string]string
}

func (r *image) initResult(numItems int) {
	r.result = make(map[string]*Output, len(r.outputContext))
	r.numItems = numItems
}

func (r *image) recordResult(o any) {
	r.m.Lock()
	defer r.m.Unlock()

	rctx, ok := o.(*rctxv1.ResourceContext)
	if !ok {
		fmt.Println("unexpetec result object")
	}
	for gvkString, krmslice := range rctx.Spec.Properties.Output {
		varName := r.gvkToVarName[gvkString]
		oc := r.outputContext[varName]
		if _, ok := r.result[varName]; !ok {
			r.result[varName] = &Output{
				Internal: oc.Internal,
				Value:    make([]any, 0, r.numItems),
			}
		}

		for _, krm := range krmslice {
			x := map[string]any{}
			if err := json.Unmarshal([]byte(krm), &x); err != nil {
				fmt.Printf("error unmarshaling the data, err: %s\n", err.Error())
			}

			switch r.result[varName].Value.(type) {
			case []any:
				r.result[varName].Value = append(r.result[varName].Value.([]any), x)
			}
		}
	}
}

func (r *image) getResult() map[string]*Output {
	return r.result
}

func (r *image) prepareInput(fnconfig *ctrlcfgv1.Function) any {
	return fnconfig
}

func (r *image) runImage(ctx context.Context, req ctrl.Request, extraInput any, input map[string]any) (any, error) {
	fnconfig, ok := extraInput.(*ctrlcfgv1.Function)
	if !ok {
		return nil, fmt.Errorf("expecting fnconfig input, got: %T", extraInput)
	}

	runner, err := fnruntime.NewRunner(ctx, fnconfig,
		fnruntime.RunnerOptions{
			ResolveToImage: fnruntime.ResolveToImageForCLI,
		},
	)
	if err != nil {
		return nil, err
	}
	rctx, err := buildResourceContext(req, input)
	if err != nil {
		return nil, err
	}
	o, err := runner.Run(rctx)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func buildResourceContext(req ctrl.Request, input map[string]any) (*rctxv1.ResourceContext, error) {
	props, err := buildResourceContextProperties(input)
	if err != nil {
		return nil, err
	}

	rctx := &rctxv1.ResourceContext{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ResourceContext",
			APIVersion: "lcnc.yndd.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: rctxv1.ResourceContextSpec{
			Properties: props,
		},
	}

	gvk := schema.GroupVersionKind{
		Group:   "lcnc.yndd.io",
		Version: "v1",
		Kind:    "ResourceContext",
	}

	rctx.SetGroupVersionKind(gvk)
	return rctx, nil
}

func buildResourceContextProperties(input map[string]any) (*rctxv1.ResourceContextProperties, error) {
	props := &rctxv1.ResourceContextProperties{
		Origin: map[string]rctxv1.KRMResource{},
		Input:  map[string][]rctxv1.KRMResource{},
		Output: map[string][]rctxv1.KRMResource{},
	}
	for _, v := range input {
		switch x := v.(type) {
		case map[string]any:
			// we should only have 1 resource with this type which is the origin
			gvk, res, err := getGVKResource(x)
			if err != nil {
				return nil, err
			}
			props.Origin[meta.GVKToString(gvk)] = rctxv1.KRMResource(res)
		case []any:
			l := len(x)
			if l > 0 {
				for _, v := range x {
					switch x := v.(type) {
					case map[string]any:
						gvk, res, err := getGVKResource(x)
						if err != nil {
							return nil, err
						}
						if _, ok := props.Input[meta.GVKToString(gvk)]; !ok {
							props.Input[gvk.String()] = make([]rctxv1.KRMResource, 0, l)
						}
						props.Input[meta.GVKToString(gvk)] = append(props.Input[meta.GVKToString(gvk)], rctxv1.KRMResource(res))
					default:
						return nil, fmt.Errorf("unexpected object in []any: got %T", v)
					}
				}
			}
		default:
			return nil, fmt.Errorf("unexpected input object: got %T", v)
		}
	}
	return props, nil
}

func getGVKResource(x map[string]any) (*schema.GroupVersionKind, string, error) {
	apiVersion, ok := x["apiVersion"]
	if !ok {
		return nil, "", fmt.Errorf("origin is not a KRM resource apiVersion missing")
	}
	kind, ok := x["kind"]
	if !ok {
		return nil, "", fmt.Errorf("origin is not a KRM resource kind missing")
	}
	gv, err := schema.ParseGroupVersion(apiVersion.(string))
	if err != nil {
		return nil, "", err
	}
	b, err := json.Marshal(x)
	if err != nil {
		return nil, "", err
	}
	return &schema.GroupVersionKind{
			Group:   gv.Group,
			Version: gv.Version,
			Kind:    kind.(string)},
		string(b), nil
}
