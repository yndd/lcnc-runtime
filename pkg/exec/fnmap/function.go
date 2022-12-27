package fnmap

import (
	"context"

	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Function interface {
	// Init initializes the device
	Init(...FunctionOption)
	WithOutput(output output.Output)
	WithResult(result result.Result)
	WithNameAndNamespace(name, namespace string)
	WithClient(client client.Client)
	WithFnMap(fnMap FuncMap)
	Run(ctx context.Context, vertexContext *rtdag.VertexContext, i input.Input) (output.Output, error)
}

type FunctionOption func(Function)

func WithOutput(output output.Output) FunctionOption {
	return func(r Function) {
		r.WithOutput(output)
	}
}

func WithResult(result result.Result) FunctionOption {
	return func(r Function) {
		r.WithResult(result)
	}
}

func WithNameAndNamespace(name, namespace string) FunctionOption {
	return func(r Function) {
		r.WithNameAndNamespace(name, namespace)
	}
}

func WithClient(client client.Client) FunctionOption {
	return func(r Function) {
		r.WithClient(client)
	}
}

func WithFnMap(fnMap FuncMap) FunctionOption {
	return func(r Function) {
		r.WithFnMap(fnMap)
	}
}
