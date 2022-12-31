package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/yndd/lcnc-function-sdk/go/fn"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnruntime"
	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		filterInputFn: r.filterInput,
		runFn:         r.run,
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
	name           string
	namespace      string
	rootVertexName string
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

func (r *image) WithRootVertexName(name string) {
	r.rootVertexName = name
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
	rctx, ok := o.(*fn.ResourceContext)
	if !ok {
		err := fmt.Errorf("expected type *rctxv1.ResourceContext, got: %T", o)
		r.l.Error(err, "cannot record output")
		r.errs = append(r.errs, err.Error())
		return
	}
	for gvkString, krmslice := range rctx.Resources.Output {
		varName, ok := r.gvkToVarName[gvkString]
		if !ok {
			err := fmt.Errorf("unregistered image output gvk: %s", gvkString)
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

// for the image we filter the input
// we convert to resourceContext and this might fail if we provide
// unneccessary variables
func (r *image) filterInput(i input.Input) input.Input {
	newInput := input.New()
	for varName, v := range i.Get() {
		for ivarName := range r.fnconfig.Vars {
			switch {
			case varName == ivarName:
				newInput.AddEntry(varName, v)
			case varName == r.rootVertexName:
				newInput.AddEntry(varName, v)
			}
		}
	}
	return newInput
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
	rCtx, err := buildResourceContext(i)
	if err != nil {
		r.l.Error(err, "cannot build resource context")
		return nil, err
	}
	o, err := runner.Run(rCtx)
	if err != nil {
		r.l.Error(err, "failed tunner")
		return nil, err
	}
	return o, nil
}

func buildResourceContext(i input.Input) (*fn.ResourceContext, error) {
	resources, err := buildResourceContextResources(i)
	if err != nil {
		return nil, err
	}

	rCtx := &fn.ResourceContext{
		Resources: resources,
	}

	return rCtx, nil
}

func buildResourceContextResources(i input.Input) (*fn.Resources, error) {
	resources := &fn.Resources{
		Input:      map[string][]runtime.RawExtension{},
		Output:     map[string][]runtime.RawExtension{},
		Conditions: map[string][]runtime.RawExtension{},
	}
	i.Print("runImage")
	for _, v := range i.Get() {
		switch x := v.(type) {
		case map[string]any:
			o, err := getObject(x)
			if err != nil {
				return nil, err
			}
			if err := resources.AddUniqueIntput(o); err != nil {
				return nil, err
			}
		case []any:
			l := len(x)
			if l > 0 {
				for _, v := range x {
					switch x := v.(type) {
					case map[string]any:
						o, err := getObject(x)
						if err != nil {
							return nil, err
						}
						if err := resources.AddUniqueIntput(o); err != nil {
							return nil, err
						}
					case []any:
						l := len(x)
						if l > 0 {
							for _, v := range x {
								switch x := v.(type) {
								case map[string]any:
									o, err := getObject(x)
									if err != nil {
										return nil, err
									}
									if err := resources.AddUniqueIntput(o); err != nil {
										return nil, err
									}
								default:
									return nil, fmt.Errorf("unexpected object in []any[]any: got %T", v)
								}
							}
						}
					default:
						return nil, fmt.Errorf("unexpected object in []any: got %T", v)
					}
				}
			}
		default:
			return nil, fmt.Errorf("unexpected input object: got %T", v)
		}
	}
	return resources, nil
}

func getObject(x map[string]any) (fn.Object, error) {
	b, err := json.Marshal(x)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	if err := json.Unmarshal(b, u); err != nil {
		return nil, err
	}
	return u, nil
}
