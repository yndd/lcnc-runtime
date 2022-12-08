package fnmap

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

type FnMap interface {
}

func New(c client.Client) FnMap {
	f := &fnMap{
		client: c,
		fns:    map[string]interface{}{},
	}
	f.fns["queryClient"] = f.queryGvk

	return f
}

type fnMap struct {
	client client.Client
	fns    map[string]interface{}
}


func (r *fnMap) queryGvk(ctx context.Context, namespace string, gvk schema.GroupVersionKind, ccquery ctrlcfgv1.ControllerConfigQuery) ([]unstructured.Unstructured, error) {
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(ccquery.Selector.MatchLabels),
	}

	
	crl := getUnstructuredList(gvk)
	if err := r.client.List(ctx, crl, opts...); err != nil {
		return nil, err
	}
	return crl.Items, nil

}

func forSlice() {
	s := make([]interface)
}

func getUnstructuredList(gvk schema.GroupVersionKind) *unstructured.UnstructuredList {
	var u unstructured.UnstructuredList
	u.SetAPIVersion(gvk.GroupVersion().String())
	u.SetKind(gvk.Kind)
	uCopy := u.DeepCopy()
	return uCopy
}
