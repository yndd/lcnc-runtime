package v1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   Spec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status Status `json:"status,omitempty" yaml:"status,omitempty"`
}

type Spec struct {
	Properties *Properties `json:"properties,omitempty"`
}

// ResourceContextSpec defines the context of the resource of the controller
type Properties struct {
	// holds the input of the CR
	Origin map[string]runtime.RawExtension `json:"origin,omitempty" yaml:"origin,omitempty"`
	// holds the input of the CR
	Input map[string][]runtime.RawExtension `json:"input,omitempty" yaml:"input,omitempty"`
	// holds the allocation of the CR with the key being GVK in string format
	Conditions map[string][]runtime.RawExtension `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	// holds the output of the CR with the key being GVK in string format
	Output map[string][]runtime.RawExtension `json:"output,omitempty" yaml:"output,omitempty"`
	//Result[]
}

// Status defines the context of the resource of the controller
type Status struct {
}

// ResourceContextList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
type ResourceContextList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Items []ResourceContext `json:"items" yaml:"items"`
}
