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
	//Functions map[string]ControllerConfigFunctionBlock `json:",inline" yaml:",inline"`

	//Vars      []ControllerConfigVarBlock       `json:"vars,omitempty" yaml:"vars,omitempty"`
	//Functions []ControllerConfigFunctionsBlock `json:"fucntions,omitempty" yaml:"functions,omitempty"`
	//Services  []ControllerConfigFunctionsBlock `json:"services,omitempty" yaml:"services,omitempty"`
	//Services map[string]ControllerConfigFunction `json:"services,omitempty" yaml:"services,omitempty"`
}

type ControllerConfigGvrObject struct {
	Gvr                      *ControllerConfigGvr `json:"gvr" yaml:"gvr"`
	ControllerConfigPipeline `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
}

type ControllerConfigGvr struct {
	ApiVersion string `json:"apiVersion" yaml:"apiVersion"`
	Resource   string `json:"resource" yaml:"resource"`
}
type ControllerConfigPipeline struct {
	Functions map[string]ControllerConfigPipelineBlock `json:",inline" yaml:",inline"`
}

type ControllerConfigPipelineBlock struct {
	ControllerConfigBlock    `json:",inline" yaml:",inline"`
	ControllerConfigPipeline `json:",inline" yaml:",inline"`
	ControllerConfigFunction `json:",inline" yaml:",inline"`
}

type ControllerConfigBlock struct {
	ControllerConfigCondition `json:",inline" yaml:",inline"`
	ControllerConfigRange     `json:",inline" yaml:",inline"`
}

type ControllerConfigRange struct {
	Range *ControllerConfigRangeValue `json:"range,omitempty" yaml:"range,omitempty"`
}

type ControllerConfigRangeValue struct {
	Value                 *string `json:"value,omitempty" yaml:"value,omitempty"`
	ControllerConfigBlock `json:",inline" yaml:",inline"`
}

type ControllerConfigCondition struct {
	Condition *ControllerConfigConditionExpression `json:"condition,omitempty" yaml:"condition,omitempty"`
}

type ControllerConfigConditionExpression struct {
	Expression            *string `json:"expression" yaml:"expression"`
	ControllerConfigBlock `json:",inline" yaml:",inline"`
}

type ControllerConfigFunction struct {
	ControllerConfigExecutor `json:",inline" yaml:",inline"`
	Functions                map[string]ControllerConfigPipelineBlock `json:",inline" yaml:",inline"`
	Type                     string                                   `json:"type,omitempty" yaml:"type,omitempty"`
	//Config                   string                                   `json:"config,omitempty" yaml:"config,omitempty"`
	// input is always a GVK of some sort
	Input *ControllerConfigInput `json:"input,omitempty" yaml:"input,omitempty"`
	// key = variableName, value is gvr format or not -> gvr format is needed for external resources
	Output map[string]string `json:"output,omitempty" yaml:"output,omitempty"`
}

type ControllerConfigInput struct {
	Gvr                          *ControllerConfigGvr  `json:"gvr,omitempty" yaml:"gvr,omitempty"`
	Selector                     *metav1.LabelSelector `json:"selector,omitempty"`
	Key                          string                `json:"key,omitempty" yaml:"key,omitempty"`
	Value                        string                `json:"value,omitempty" yaml:"value,omitempty"`
	ControllerConfigGenericInput map[string]string     `json:",inline" yaml:",inline"`
}

type ControllerConfigExecutor struct {
	Image *string `json:"image,omitempty" yaml:"image,omitempty"`
	Exec  *string `json:"exec,omitempty" yaml:"exec,omitempty"`
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
