package fnmap

import (
	"context"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FnMap interface {
	RunFn(ctx context.Context, req ctrl.Request, vertexContext *dag.VertexContext, input map[string]any) (map[string]*Output, error)
}

type FnMapConfig struct {
	Client client.Client
	GVK    *schema.GroupVersionKind
}

func New(fmc *FnMapConfig) FnMap {
	f := &fnmap{
		client: fmc.Client,
		gvk:    fmc.GVK,
	}
	return f
}

type fnmap struct {
	client client.Client
	gvk    *schema.GroupVersionKind
}

type Result map[string]*Output

type Output struct {
	Internal bool
	Value    any
}
