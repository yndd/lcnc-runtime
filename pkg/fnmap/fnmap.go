package fnmap

import (
	"context"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FnMap interface {
	RunFn(ctx context.Context, fnconfig *ctrlcfgv1.ControllerConfigFunction, input map[string]any) (any, error)
}

func New(c client.Client, gvk schema.GroupVersionKind) FnMap {
	f := &fnmap{
		client: c,
		gvk:    gvk,
		//fns:    map[string]interface{}{},
	}
	//f.fns["queryClient"] = f.queryGvk

	return f
}

type fnmap struct {
	client client.Client
	gvk    schema.GroupVersionKind
	//fns    map[string]interface{}
}
