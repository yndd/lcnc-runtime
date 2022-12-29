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
	// fnconfig provides additional configuration for the function
	FunctionConfig runtime.RawExtension `json:"functionConfig,omitempty" yaml:"functionConfig,omitempty"`
	// holds the output of the CR with the key being GVK in string format
	Output map[string][]runtime.RawExtension `json:"output,omitempty" yaml:"output,omitempty"`
	// holds the allocation of the CR with the key being GVK in string format
	Conditions map[string][]runtime.RawExtension `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	// results provide a structured
	Results Results `json:"results,omitempty" yaml:"results,omitempty"`
}

type Results []*Result

// Result defines a result for the fucntion execution
type Result struct {
	// Message is a human readable message. This field is required.
	Message string `json:"message,omitempty" yaml:"message,omitempty"`

	// ResourceRef is a reference to a resource.
	// Required fields: apiVersion, kind, name.
	ResourceRef *ResourceRef `json:"resourceRef,omitempty" yaml:"resourceRef,omitempty"`
}

// ResourceRef fills the ResourceRef field in Results
type ResourceRef struct {
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty" yaml:"kind,omitempty"`
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace  string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
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
