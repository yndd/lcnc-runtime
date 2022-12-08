package v1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	ControllerConfigKind     = "ControllerConfig"
	ControllerConfigListKind = "ControllerConfigList"

	ControllerConfigSingular  = "controllerconfig"
	ControllerConfigPlural    = "controllerconfigs"
	ControllerConfigShortName = "ccfg"
)

var (
	GroupVersion                    = schema.GroupVersion{Group: Group, Version: Version}
	ResourceContextGroupKind        = schema.GroupKind{Group: Group, Kind: ControllerConfigKind}.String()
	ResourceContextKindAPIVersion   = ControllerConfigKind + "." + GroupVersion.String()
	ResourceContextGroupVersionKind = GroupVersion.WithKind(ControllerConfigKind)

	ControllerPkgRevLabelKey = strings.ToLower(ControllerConfigKind) + "/" + "PackageRevision"
)

// ControllerConfig is the Schema for the ControllerConfig controller API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ControllerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControllerConfigSpec   `json:"spec,omitempty"`
	Status ControllerConfigStatus `json:"status,omitempty"`
}

type ControllerConfigSpec struct {
	Properties *ControllerConfigProperties `json:"properties,omitempty"`
}

type ControllerConfigProperties struct {
	// key represents the variable
	For map[string]ControllerConfigGvrObject `json:"for" yaml:"for"`
	// key represents the variable
	Own map[string]ControllerConfigGvrObject `json:"own,omitempty" yaml:"own,omitempty"`
	// key represents the variable
	Watch map[string]ControllerConfigGvrObject `json:"watch,omitempty" yaml:"watch,omitempty"`
	// key respresents the variable
	Vars      []ControllerConfigVarBlock       `json:"vars,omitempty" yaml:"vars,omitempty"`
	Functions []ControllerConfigFunctionsBlock `json:"fucntions,omitempty" yaml:"functions,omitempty"`
	Services  []ControllerConfigFunctionsBlock `json:"services,omitempty" yaml:"services,omitempty"`
	//Services map[string]ControllerConfigFunction `json:"services,omitempty" yaml:"services,omitempty"`
}

type ControllerConfigGvrObject struct {
	Gvr                      string `json:"gvr" yaml:"gvr"`
	ControllerConfigExecutor `json:",inline" yaml:",inline"`
}

type ControllerConfigVarBlock struct {
	ControllerConfigBlock     `json:",inline" yaml:",inline"`
	ControllerConfigVariables map[string]ControllerConfigVar `json:",inline" yaml:",inline"`
}

type ControllerConfigBlock struct {
	For *ControllerConfigFor `json:"for,omitempty" yaml:"for,omitempty"`
	// TODO add IF statement block as standalone and within the if statement
}

type ControllerConfigFor struct {
	Range *string `json:"range,omitempty" yaml:"range,omitempty"`
}

type ControllerConfigVar struct {
	Slice *ControllerConfigSlice `json:"slice,omitempty" yaml:"slice,omitempty"`
	Map   *ControllerConfigMap   `json:"map,omitempty" yaml:"map,omitempty"`
}

type ControllerConfigSlice struct {
	ControllerConfigValue `json:"value,omitempty" yaml:"value,omitempty"`
}

type ControllerConfigMap struct {
	Key                   *string `json:"key,omitempty" yaml:"key,omitempty"`
	ControllerConfigValue `json:"value,omitempty" yaml:"value,omitempty"`
}

type ControllerConfigValue struct {
	ControllerConfigQuery `json:",inline" yaml:",inline"`
	String                *string `json:"string,omitempty" yaml:"string,omitempty"`
}

type ControllerConfigFunctionsBlock struct {
	ControllerConfigBlock     `json:",inline" yaml:",inline"`
	ControllerConfigFunctions map[string]ControllerConfigFunction `json:",inline" yaml:",inline"`
}

type ControllerConfigFunction struct {
	ControllerConfigExecutor `json:",inline" yaml:",inline"`
	//Vars      []ControllerConfigVarBlock    `json:"vars,omitempty" yaml:"vars,omitempty"`
	Vars   map[string]ControllerConfigVar `json:"vars,omitempty" yaml:"vars,omitempty"`
	Config string                         `json:"config,omitempty" yaml:"config,omitempty"`
	// input is always a GVK of some sort
	Input map[string]string `json:"input,omitempty" yaml:"input,omitempty"`
	// key = variableName, value is gvr format or not -> gvr format is needed for external resources
	Output map[string]string `json:"output,omitempty" yaml:"output,omitempty"`
}

type ControllerConfigExecutor struct {
	Image *string `json:"image,omitempty" yaml:"image,omitempty"`
	Exec  *string `json:"exec,omitempty" yaml:"exec,omitempty"`
}

type ControllerConfigQuery struct {
	Query    *string                   `json:"query,omitempty" yaml:"query,omitempty"`
	Selector *ControllerConfigSelector `json:"selector,omitempty" yaml:"selector,omitempty"`
}

type ControllerConfigSelector struct {
	Name        *string           `json:"name,omitempty" yaml:"name,omitempty"`
	MatchLabels map[string]string `json:"matchLabels,omitempty" yaml:"matchLabels,omitempty"`
}

// ResourceContextSpec defines the context of the resource of the controller
type ControllerConfigStatus struct {
}

// ControllerConfigList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
type ControllerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ControllerConfig `json:"items"`
}

/*
type ControllerConfig struct {
	Name string `yaml:"image,omitempty" json:"image,omitempty"`
	//Mgr              manager.Manager
	//GlobalPredicates []predicate.Predicate
	For   *ControllerPipeline   `yaml:"for" json:"for"`
	Owns  []*ControllerPipeline `yaml:"owns,omitempty" json:"owns,omitempty"`
	Watch []*ControllerPipeline `yaml:"watch,omitempty" json:"watch,omitempty"`
	//Ctrl             controller.Controller
	//CtrlOptions      controller.Options
}

type ControllerPipeline struct {
	Gvr *ControllerGroupVersionResource `yaml:"gvr" json:"gvr"`
	Fn  []*Function                       `yaml:"function" json:"function"`
	//Predicates   []predicate.Predicate
	Eventhandler handler.EventHandler
}

type ControllerGroupVersionResource struct {
	Group    string `yaml:"group,omitempty" json:"group,omitempty"`
	Version  string `yaml:"version,omitempty" json:"version,omitempty"`
	Resource string `yaml:"resurce,omitempty" json:"resource,omitempty"`
}

type Function struct {
	// `Image` specifies the function container image.
	// It can either be fully qualified, e.g.:
	//
	//    image: docker.io/set-topology
	//
	// Optionally, kpt can be configured to use a image
	// registry host-path that will be used to resolve the image path in case
	// the image path is missing (Defaults to docker.io/yndd).
	// e.g. The following resolves to docker.io/yndd/set-topology:
	//
	//    image: set-topology
	Image string `yaml:"image,omitempty" json:"image,omitempty"`

	// Exec specifies the function binary executable.
	// The executable can be fully qualified or it must exists in the $PATH e.g:
	//
	//      exec: set-topology
	//      exec: /usr/local/bin/my-custom-fn
	Exec string `yaml:"exec,omitempty" json:"exec,omitempty"`
}
*/
