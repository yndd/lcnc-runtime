package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	rctxv1 "github.com/yndd/lcnc-runtime/pkg/api/resourcecontext/v1"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnruntime"
	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func NewImageFn() fnmap.Function {
	l := ctrl.Log.WithName("image fn")
	r := &image{
		errs: make([]string, 0),
		l:    l,
	}

	r.fec = &fnExecConfig{
		executeRange:  true,
		executeSingle: true,
		// execution functions
		runFn: r.run,
		// result functions
		initOutputFn:     r.initOutput,
		recordOutputFn:   r.recordOutput,
		getFinalResultFn: r.getFinalResult,
		l:                l,
	}
	return r
}

type image struct {
	// fec exec config
	fec *fnExecConfig
	// init config
	name      string
	namespace string
	// runtime config
	fnconfig     *ctrlcfgv1.Function
	outputs      output.Output
	gvkToVarName map[string]string
	// result, output
	m        sync.RWMutex
	output   output.Output
	numItems int
	errs     []string
	// logging
	l logr.Logger
}

func (r *image) Init(opts ...fnmap.FunctionOption) {
	for _, o := range opts {
		o(r)
	}
}

func (r *image) WithOutput(output output.Output) {}

func (r *image) WithResult(result result.Result) {}

func (r *image) WithNameAndNamespace(name, namespace string) {
	r.name = name
	r.namespace = namespace
}

func (r *image) WithClient(client client.Client) {}

func (r *image) WithFnMap(fnMap fnmap.FuncMap) {}

func (r *image) Run(ctx context.Context, vertexContext *rtdag.VertexContext, i input.Input) (output.Output, error) {
	r.l.Info("run", "vertexName", vertexContext.VertexName, "input", i.Get(), "gvkToName", vertexContext.GVKToVarName)

	// Here we prepare the input we get from the runtime
	// e.g. DAG, outputs/outputInfo (internal/GVK/etc), fnConfig parameters, etc etc
	r.fnconfig = vertexContext.Function
	r.outputs = vertexContext.Outputs
	r.gvkToVarName = vertexContext.GVKToVarName

	// execute the function
	return r.fec.exec(ctx, vertexContext.Function, i)
}

func (r *image) initOutput(numItems int) {
	r.output = output.New()
	r.numItems = numItems
}

func (r *image) recordOutput(o any) {
	r.m.Lock()
	defer r.m.Unlock()
	rctx, ok := o.(*rctxv1.ResourceContext)
	if !ok {
		err := fmt.Errorf("expected type *rctxv1.ResourceContext, got: %T", o)
		r.l.Error(err, "cannot record output")
		r.errs = append(r.errs, err.Error())
		return
	}
	for gvkString, krmslice := range rctx.Spec.Properties.Output {
		varName, ok := r.gvkToVarName[gvkString]
		if !ok {
			err := fmt.Errorf("unregistered image gvk: %s", gvkString)
			r.l.Error(err, "cannot record output since the variable is not initialized")
			r.errs = append(r.errs, err.Error())
			break
		}

		krmOutput := make([]any, 0, len(krmslice))
		for _, krm := range krmslice {
			x := map[string]any{}
			if err := json.Unmarshal(krm.Raw, &x); err != nil {
				r.l.Error(err, "cannot unmarshal data")
				r.errs = append(r.errs, err.Error())
				break
			}

			krmOutput = append(krmOutput, x)
		}

		v, ok := r.outputs.Get()[varName]
		if !ok {
			err := fmt.Errorf("unregistered image varName: %s", varName)
			r.l.Error(err, "cannot record output")
			r.errs = append(r.errs, err.Error())
			break
		}
		oi, ok := v.(*output.OutputInfo)
		if !ok {
			err := fmt.Errorf("expected type *output.OutputInfo, got: %T", v)
			r.l.Error(err, "cannot record output")
			r.errs = append(r.errs, err.Error())
			break
		}
		r.output.AddEntry(varName, &output.OutputInfo{
			Internal: oi.Internal,
			GVK:      oi.GVK,
			Data:     krmOutput,
		})
	}
}

func (r *image) getFinalResult() (output.Output, error) {
	if len(r.errs) > 0 {
		return nil, fmt.Errorf("errors executing image: %v", r.errs)
	}
	return r.output, nil
}

func (r *image) run(ctx context.Context, i input.Input) (any, error) {
	runner, err := fnruntime.NewRunner(ctx, r.fnconfig,
		fnruntime.RunnerOptions{
			ResolveToImage: fnruntime.ResolveToImageForCLI,
		},
	)
	if err != nil {
		r.l.Error(err, "cannot get runner")
		return nil, err
	}
	rctx, err := buildResourceContext(r.name, r.namespace, i)
	if err != nil {
		r.l.Error(err, "cannot build resource context")
		return nil, err
	}
	o, err := runner.Run(rctx)
	if err != nil {
		r.l.Error(err, "failed tunner")
		return nil, err
	}
	return o, nil
}

func buildResourceContext(name, namespace string, i input.Input) (*rctxv1.ResourceContext, error) {
	props, err := buildResourceContextProperties(i)
	if err != nil {
		return nil, err
	}

	rctx := &rctxv1.ResourceContext{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ResourceContext",
			APIVersion: "lcnc.yndd.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: rctxv1.Spec{
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

func buildResourceContextProperties(i input.Input) (*rctxv1.Properties, error) {
	props := &rctxv1.Properties{
		Origin: map[string]runtime.RawExtension{},
		Input:  map[string][]runtime.RawExtension{},
		Output: map[string][]runtime.RawExtension{},
	}
	for _, v := range i.Get() {
		switch x := v.(type) {
		case map[string]any:
			// we should only have 1 resource with this type which is the origin
			gvk, res, err := getGVKResource(x)
			if err != nil {
				return nil, err
			}
			props.Origin[meta.GVKToString(gvk)] = runtime.RawExtension{Raw: []byte(res)}
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
							props.Input[gvk.String()] = make([]runtime.RawExtension, 0, l)
						}
						props.Input[meta.GVKToString(gvk)] = append(props.Input[meta.GVKToString(gvk)], runtime.RawExtension{Raw: []byte(res)})
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
	b, err := yaml.Marshal(x)
	if err != nil {
		return nil, "", err
	}
	return &schema.GroupVersionKind{
			Group:   gv.Group,
			Version: gv.Version,
			Kind:    kind.(string)},
		string(b), nil
}
