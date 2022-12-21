package v1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   Spec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status Status `json:"status,omitempty" yaml:"status,omitempty"`
}

type Spec struct {
	Properties *Properties `json:"properties,omitempty" yaml:"properties,omitempty"`
}

type Properties struct {
	// key represents the variable
	For map[string]*GvkObject `json:"for" yaml:"for"`
	// key represents the variable
	Own map[string]*GvkObject `json:"own,omitempty" yaml:"own,omitempty"`
	// key represents the variable
	Watch map[string]*GvkObject `json:"watch,omitempty" yaml:"watch,omitempty"`
	// key respresents the variable
	//Functions map[string]ControllerConfigFunctionBlock `json:",inline" yaml:",inline"`
	Pipelines []*Pipeline `json:"pipelines,omitempty" yaml:"pipelines,omitempty"`

	//Services  []ControllerConfigFunctionsBlock `json:"services,omitempty" yaml:"services,omitempty"`
	//Services map[string]ControllerConfigFunction `json:"services,omitempty" yaml:"services,omitempty"`
}

type GvkObject struct {
	Resource    runtime.RawExtension `json:"resource,omitempty" yaml:"resource,omitempty"`
	PipelineRef string               `json:"pipelineRef,omitempty" yaml:"pipelineRef,omitempty"`
}

type Pipeline struct {
	Name  string               `json:"name" yaml:"name"`
	Vars  map[string]*Function `json:"vars,omitempty" yaml:"vars,omitempty"`
	Tasks map[string]*Function `json:"tasks,omitempty" yaml:"tasks,omitempty"`
}

type Block struct {
	Range     *RangeValue          `json:"range,omitempty" yaml:"range,omitempty"`
	Condition *ConditionExpression `json:"condition,omitempty" yaml:"condition,omitempty"`
}

type RangeValue struct {
	Value string `json:"value" yaml:"value"`
	Block `json:",inline" yaml:",inline"`
	//Range     *RangeValue          `json:"range,omitempty" yaml:"range,omitempty"`
	//Condition *ConditionExpression `json:"condition,omitempty" yaml:"condition,omitempty"`
}

type ConditionExpression struct {
	Expression string `json:"expression" yaml:"expression"`
	Block      `json:",inline" yaml:",inline"`
	//Range      *RangeValue          `json:"range,omitempty" yaml:"range,omitempty"`
	//Condition  *ConditionExpression `json:"condition,omitempty" yaml:"condition,omitempty"`
}

type FunctionType string

const (
	ForQueryType FunctionType = "forQuery"
	QueryType    FunctionType = "query"
	SliceType    FunctionType = "slice"
	MapType      FunctionType = "map"
	JQType       FunctionType = "jq"
	Container    FunctionType = "container"
	Wasm         FunctionType = "wasm"
	GoTemplate   FunctionType = "gotemplate"
)

type Function struct {
	Block    `json:",inline" yaml:",inline"`
	Executor `json:",inline" yaml:",inline"`
	// Vars define the local variables in the function
	// The Key respresents the local variable name
	// The Value represents the jq expression
	Vars   map[string]string `json:"vars,omitempty" yaml:"vars,omitempty"`
	Type   FunctionType      `json:"type,omitempty" yaml:"type,omitempty"`
	Config string            `json:"config,omitempty" yaml:"config,omitempty"`
	// input is always a GVK of some sort
	Input *Input `json:"input,omitempty" yaml:"input,omitempty"`
	// key = variableName, value is gvr format or not -> gvr format is needed for external resources
	Output    map[string]*Output `json:"output,omitempty" yaml:"output,omitempty"`
	DependsOn []string           `json:"dependsOn,omitempty" yaml:"dependsOn,omitempty"`
}

type Output struct {
	Internal bool                 `json:"internal" yaml:"internal"`
	Resource runtime.RawExtension `json:"resource" yaml:"resource"`
	//GenericOutput string               `json:",inline" yaml:",inline"`
}

type Input struct {
	Selector     *metav1.LabelSelector `json:"selector,omitempty" yaml:"selector,omitempty"`
	Key          string                `json:"key,omitempty" yaml:"key,omitempty"`
	Value        string                `json:"value,omitempty" yaml:"value,omitempty"`
	GenericInput map[string]string     `json:",inline" yaml:",inline"`
	Expression   string                `json:"expression,omitempty" yaml:"expression,omitempty"`
	Resource     runtime.RawExtension  `json:"resource,omitempty" yaml:"resource,omitempty"`
	Template     string                `json:"template,omitempty" yaml:"template,omitempty"`
}

type Executor struct {
	Image *string `json:"image,omitempty" yaml:"image,omitempty"`
	Exec  *string `json:"exec,omitempty" yaml:"exec,omitempty"`
}

// ResourceContextSpec defines the context of the resource of the controller
type Status struct {
}

// ControllerConfigList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
type ControllerConfigList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Items []ControllerConfig `json:"items" yaml:"items"`
}
