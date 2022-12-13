package fnmap

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FnMap interface {
}

func New(c client.Client) FnMap {
	f := &fnMap{
		client: c,
		fns:    map[string]interface{}{},
	}
	//f.fns["queryClient"] = f.queryGvk

	return f
}

type fnMap struct {
	client client.Client
	fns    map[string]interface{}
}

/*
func (r *fnMap) queryGvk(ctx context.Context, namespace string, fn ctrlcfgv1.ControllerConfigFunction) ([]unstructured.Unstructured, error) {
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(fn.Input.Selector.MatchLabels),
	}


	crl := getUnstructuredList(fn.Input.Gvr)
	if err := r.client.List(ctx, crl, opts...); err != nil {
		return nil, err
	}
	return crl.Items, nil

}
*/

/*
func forSlice() {
	s := make([]interface)
}
*/

func getUnstructuredList(gvk schema.GroupVersionKind) *unstructured.UnstructuredList {
	var u unstructured.UnstructuredList
	u.SetAPIVersion(gvk.GroupVersion().String())
	u.SetKind(gvk.Kind)
	uCopy := u.DeepCopy()
	return uCopy
}
