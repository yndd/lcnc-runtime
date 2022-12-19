package fnmap

import (
	"context"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *fnmap) query(ctx context.Context, fnconfig *ctrlcfgv1.Function, input map[string]any) (any, error) {
	gvk, err := ctrlcfgv1.GetGVK(fnconfig.Input.Resource)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{}
	if fnconfig.Input.Selector != nil {
		// TODO namespace
		//opts = append(opts, client.InNamespace("x"))
		opts = append(opts, client.MatchingLabels(fnconfig.Input.Selector.MatchLabels))
	}

	o := meta.GetUnstructuredListFromGVK(gvk)
	if err := r.client.List(ctx, o, opts...); err != nil {
		return nil, err
	}
	return o.Items, nil
}