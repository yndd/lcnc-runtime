package fnmap

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	o := getUnstructured(r.gvk)
	if err := r.client.Get(ctx, key, o); err != nil {
		return nil, err
	}
	return o, nil
}

func getUnstructured(gvk schema.GroupVersionKind) *unstructured.Unstructured {
	var u unstructured.Unstructured
	u.SetAPIVersion(gvk.GroupVersion().String())
	u.SetKind(gvk.Kind)
	uCopy := u.DeepCopy()
	return uCopy
}
