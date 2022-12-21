package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	rctxv1 "github.com/yndd/lcnc-runtime/pkg/api/resourcecontext/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/fnruntime"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

type execContext struct {
	vertexName     string
	rootVertexName string
	fnMap          fnmap.FnMap

	// used to handle the dependencies between the functions
	m sync.RWMutex
	// used to send fn result from the src function
	// to the dependent function
	doneChs map[string]chan bool
	// used by the dependent vertex function to rcv the result
	// of the dependent src function
	depChs map[string]chan bool

	// used to signal the vertex function is done
	// to the main walk entry
	doneFnCh chan bool

	// identifies the time the vertex got scheduled
	visited time.Time
	// identifies the time the vertex fn started
	start time.Time
	// identifies the time the vertex fn finished
	finished time.Time

	vertexContext *dag.VertexContext

	// callback
	recordResult ResultFunc
	// output
	output Output
}

func (r *execContext) AddDoneCh(n string, c chan bool) {
	r.m.Lock()
	defer r.m.Unlock()
	r.doneChs[n] = c
}

func (r *execContext) AddDepCh(n string, c chan bool) {
	r.m.Lock()
	defer r.m.Unlock()
	r.depChs[n] = c
}

func (r *execContext) isFinished() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return !r.finished.IsZero()
}

func (r *execContext) isVisted() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return !r.visited.IsZero()
}

func (r *execContext) run(ctx context.Context, req ctrl.Request) {
	r.m.Lock()
	r.start = time.Now()
	r.m.Unlock()

	// Gather the input based on the function type
	// Determine if this is an internal fn runner or not
	input := map[string]any{}
	internalRunner := true
	switch r.vertexContext.Function.Type {
	case ctrlcfgv1.Container, ctrlcfgv1.Wasm:
		input[r.rootVertexName] = r.output.Get(r.rootVertexName)
		for _, ref := range r.vertexContext.References {
			input[ref] = r.output.Get(ref)
		}
		internalRunner = false
	case ctrlcfgv1.ForQueryType:
		// we use a dedicated key for the for
		input[fnmap.ForKey] = req.NamespacedName
	default:
		fmt.Printf("references: %v\n", r.vertexContext.References)
		for _, ref := range r.vertexContext.References {
			input[ref] = r.output.Get(ref)
		}
	}
	fmt.Printf("vertex: %s input: %#v\n", r.vertexName, input)

	// Run the execution context
	success := true
	reason := ""
	var o any //outout
	if internalRunner {
		var err error
		o, err = r.fnMap.RunFn(ctx, r.vertexContext.Function, input)
		if err != nil {
			if !errors.Is(err, fnmap.ErrConditionFalse) {
				success = false
			}
			reason = err.Error()
		}
	} else {
		runner, err := fnruntime.NewRunner(ctx, r.vertexContext.Function,
			fnruntime.RunnerOptions{
				ResolveToImage: fnruntime.ResolveToImageForCLI,
			},
		)
		if err != nil {
			success = false
			reason = err.Error()
		}
		rctx, err := buildResourceContext(req, input)
		if err != nil {
			success = false
			reason = err.Error()
		}
		o, err := runner.Run(rctx)
		if err != nil {
			success = false
			reason = err.Error()
		}
		// o is a resourceContext with output
		fmt.Printf("rctx: %v\n", o)
	}

	fmt.Printf("vertex: %s, success: %t, reason: %s, output: %v\n", r.vertexName, success, reason, o)

	fmt.Printf("%s fn executed, doneChs: %v\n", r.vertexName, r.doneChs)
	r.m.Lock()
	r.finished = time.Now()
	r.m.Unlock()

	// callback function to capture the result
	r.recordResult(&result{
		vertexName: r.vertexName,
		startTime:  r.start,
		endTime:    r.finished,
		outputCtx:  r.vertexContext.OutputContext,
		output:     o,
		success:    success,
		reason:     reason,
	})

	// signal to the dependent function the result of the vertex fn execution
	for vertexName, doneCh := range r.doneChs {
		doneCh <- success
		close(doneCh)
		fmt.Printf("%s -> %s send done\n", r.vertexName, vertexName)
	}
	// signal the result of the vertex execution to the main walk
	r.doneFnCh <- success
	close(r.doneFnCh)
	fmt.Printf("%s -> walk main fn done\n", r.vertexName)
}

func (r *execContext) waitDependencies(ctx context.Context) bool {
	// for each dependency wait till a it completed, either through
	// the dependency Channel or cancel or

	fmt.Printf("%s wait dependencies: %v\n", r.vertexName, r.depChs)
DepSatisfied:
	for depVertexName, depCh := range r.depChs {
		//fmt.Printf("waitDependencies %s -> %s\n", depVertexName, r.vertexName)
		//DepSatisfied:
		for {
			select {
			case d, ok := <-depCh:
				fmt.Printf("%s -> %s rcvd done, d: %t, ok: %t\n", depVertexName, r.vertexName, d, ok)
				if ok {
					continue DepSatisfied
				}
				if !d {
					// dependency failed
					return false
				}
				continue DepSatisfied
			case <-time.After(time.Second * 5):
				fmt.Printf("wait timeout vertex: %s is waiting for %s\n", r.vertexName, depVertexName)
			}
		}
	}
	fmt.Printf("%s finished waiting\n", r.vertexName)
	return true
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
