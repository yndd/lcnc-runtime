package fnmap

import (
	"context"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Function interface {
	// Init initializes the device
	Init(...FunctionOption)
	WithOutput(output output.Output)
	WithNameAndNamespace(name, namespace string)
	WithClient(client client.Client)
	WithFnMap(fnMap FuncMap)
	Run(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (output.Output, error)
}

type FunctionOption func(Function)

func WithOutput(output output.Output) FunctionOption {
	return func(r Function) {
		r.WithOutput(output)
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
