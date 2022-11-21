package v1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	ResourceContextKind     = "ResourceContext"
	ResourceContextListKind = "ResourceContextList"

	ResourceContextSingular  = "resourcecontext"
	ResourceContextPlural    = "resourcecontexts"
	ResourceContextShortName = "ctx"
)

var (
	GroupVersion                    = schema.GroupVersion{Group: Group, Version: Version}
	ResourceContextGroupKind        = schema.GroupKind{Group: Group, Kind: ResourceContextKind}.String()
	ResourceContextKindAPIVersion   = ResourceContextKind + "." + GroupVersion.String()
	ResourceContextGroupVersionKind = GroupVersion.WithKind(ResourceContextKind)

	ControllerPkgRevLabelKey = strings.ToLower(ResourceContextKind) + "/" + "PackageRevision"
)

// ResourceContext is the Schema for the lcnc runtime API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ResourceContext struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceContextSpec   `json:"spec,omitempty"`
	Status ResourceContextStatus `json:"status,omitempty"`
}

type ResourceContextSpec struct {
	Properties *ResourceContextProperties `json:"properties,omitempty"`
}

// ResourceContextSpec defines the context of the resource of the controller
type ResourceContextProperties struct {
	// holds the input of the CR
	Origin map[string]KRMResource `json:"origin,omitempty"`
	// holds the input of the CR
	Input map[string][]KRMResource `json:"input,omitempty"`
	// holds the allocation of the CR with the key being GVK in string format
	Allocations map[string][]KRMResource `json:"allocations,omitempty"`
	// holds the extra input of the CR with the key being GVK in string format
	Output map[string][]KRMResource `json:"extraInput,omitempty"`
	//Result[]
}

// string is a string representation of the KRM resource
type KRMResource string

// ResourceContextSpec defines the context of the resource of the controller
type ResourceContextStatus struct {
}

// PackageRevisionResourcesList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
type ResourceContextList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ResourceContext `json:"items"`
}
