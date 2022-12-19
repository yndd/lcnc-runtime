package fnmap

import (
	"context"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FnMap interface {
	RunFn(ctx context.Context, fnconfig *ctrlcfgv1.Function, input map[string]any) (any, error)
}

type FnMapConfig struct {
	Client client.Client
	GVK    schema.GroupVersionKind
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
	gvk    schema.GroupVersionKind
}
