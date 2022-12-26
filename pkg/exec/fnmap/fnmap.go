package fnmap

import (
	"context"
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Initializer func() Function

type FuncMap interface {
	Register(fnType ctrlcfgv1.FunctionType, initFn Initializer)
	Run(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (output.Output, error)
}

type Config struct {
	Name      string
	Namespace string
	Client    client.Client
	Output    output.Output
}

func New(c *Config) FuncMap {
	return &fnMap{
		name:      c.Name,
		namespace: c.Namespace,
		client:    c.Client,
		output:    c.Output,
		funcs:     map[ctrlcfgv1.FunctionType]Initializer{},
	}
}

type fnMap struct {
	client    client.Client
	output    output.Output
	name      string
	namespace string
	m         sync.RWMutex
	funcs     map[ctrlcfgv1.FunctionType]Initializer
}

func (r *fnMap) Register(fnType ctrlcfgv1.FunctionType, initFn Initializer) {
	r.m.Lock()
	defer r.m.Unlock()
	r.funcs[fnType] = initFn
}

func (r *fnMap) Run(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (output.Output, error) {
	r.m.RLock()
	initializer, ok := r.funcs[vertexContext.Function.Type]
	r.m.RUnlock()
	fmt.Printf("fnmap run %s, type: %s\n", vertexContext.Name, string(vertexContext.Function.Type))
	if !ok {
		return nil, fmt.Errorf("function not registered, got: %s", string(vertexContext.Function.Type))
	}
	// initialize the function
	fn := initializer()
	// initialize the runtime info
	switch vertexContext.Function.Type {
	case ctrlcfgv1.BlockType:
		fn.WithOutput(r.output)
		fn.WithFnMap(r)
	case ctrlcfgv1.QueryType:
		fn.WithClient(r.client)
	case ctrlcfgv1.ContainerType, ctrlcfgv1.WasmType:
		fn.WithNameAndNamespace(r.name, r.namespace)
	}
	// run the function
	return fn.Run(ctx, vertexContext, input)
}
