package fnmap

import (
	"context"
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Initializer func() Function

type FuncMap interface {
	Register(fnType ctrlcfgv1.FunctionType, initFn Initializer)
	Run(ctx context.Context, vertexContext *rtdag.VertexContext, i input.Input) (output.Output, error)
}

type Config struct {
	Name           string
	Namespace      string
	RootVertexName string
	Client         client.Client
	Output         output.Output
	Result         result.Result
}

func New(c *Config) FuncMap {
	return &fnMap{
		cfg:   c,
		funcs: map[ctrlcfgv1.FunctionType]Initializer{},
	}
}

type fnMap struct {
	cfg   *Config
	m     sync.RWMutex
	funcs map[ctrlcfgv1.FunctionType]Initializer
}

func (r *fnMap) Register(fnType ctrlcfgv1.FunctionType, initFn Initializer) {
	r.m.Lock()
	defer r.m.Unlock()
	r.funcs[fnType] = initFn
}

func (r *fnMap) Run(ctx context.Context, vertexContext *rtdag.VertexContext, i input.Input) (output.Output, error) {
	r.m.RLock()
	initializer, ok := r.funcs[vertexContext.Function.Type]
	r.m.RUnlock()
	//fmt.Printf("fnmap run %s, type: %s\n", vertexContext.VertexName, string(vertexContext.Function.Type))
	if !ok {
		return nil, fmt.Errorf("function not registered, got: %s", string(vertexContext.Function.Type))
	}
	// initialize the function
	fn := initializer()
	// initialize the runtime info
	switch vertexContext.Function.Type {
	case ctrlcfgv1.BlockType:
		fn.WithOutput(r.cfg.Output)
		fn.WithResult(r.cfg.Result)
		fn.WithFnMap(r)
	case ctrlcfgv1.QueryType:
		fn.WithClient(r.cfg.Client)
	case ctrlcfgv1.ContainerType, ctrlcfgv1.WasmType:
		fn.WithNameAndNamespace(r.cfg.Name, r.cfg.Namespace)
		fn.WithRootVertexName(r.cfg.RootVertexName)
	}
	// run the function
	return fn.Run(ctx, vertexContext, i)
}
