package meta

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func GetGVKfromGVR(c *rest.Config, gvr schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	mapper, err := apiutil.NewDynamicRESTMapper(c)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	gvk, err := mapper.KindFor(gvr)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return gvk, nil
}
