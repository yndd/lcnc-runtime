package fnmap

import (
	"context"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FnMap interface {
	RunFn(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (map[string]*Output, error)
}

type FnMapConfig struct {
	Name      string
	Namespace string
	Client    client.Client
	GVK       *schema.GroupVersionKind
}

func New(fmc *FnMapConfig) FnMap {
	f := &fnmap{
		name:      fmc.Name,
		namespace: fmc.Namespace,
		client:    fmc.Client,
		gvk:       fmc.GVK,
	}
	return f
}

type fnmap struct {
	name      string
	namespace string
	client    client.Client
	gvk       *schema.GroupVersionKind
}

type Result map[string]*Output

type Output struct {
	Internal bool
	Value    any
}
