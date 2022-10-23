package apiresources

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type APIResource struct {
	metav1.APIResource
	APIVersion     string
	subresourceMap map[string]bool
}

func (r *APIResource) GroupVersion() schema.GroupVersion {
	gv, err := schema.ParseGroupVersion(r.APIVersion)
	if err != nil {
		// This shouldn't happen because we get this value from discovery.
		panic(fmt.Sprintf("API discovery returned invalid group/version %q: %v", r.APIVersion, err))
	}
	return gv
}

func (r *APIResource) GroupVersionKind() schema.GroupVersionKind {
	return r.GroupVersion().WithKind(r.Kind)
}

func (r *APIResource) GroupVersionResource() schema.GroupVersionResource {
	return r.GroupVersion().WithResource(r.Name)
}

func (r *APIResource) GroupResource() schema.GroupResource {
	return schema.GroupResource{Group: r.Group, Resource: r.Name}
}

func (r *APIResource) HasSubresource(subresourceKey string) bool {
	return r.subresourceMap[subresourceKey]
}
