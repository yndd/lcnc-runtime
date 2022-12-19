package fnmap

import (
	"context"
	"fmt"

	"github.com/yndd/lcnc-runtime/pkg/meta"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ForKey = "for"
)

func (r *fnmap) forQuery(ctx context.Context, input map[string]any) (any, error) {
	// key is namespaced name
	key, ok := input[ForKey].(types.NamespacedName)
	if !ok {
		return nil, fmt.Errorf("unexpected type, expected namespacedName, got: %v", input[ForKey])
	}
	//o := getUnstructured(r.gvk)
	o := meta.GetUnstructuredFromGVK(r.gvk)
	if err := r.client.Get(ctx, key, o); err != nil {
		return nil, err
	}
	return *o, nil
}
