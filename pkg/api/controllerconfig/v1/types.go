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

	Spec   ControllerConfigSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status ControllerConfigStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

type ControllerConfigSpec struct {
	Properties *ControllerConfigProperties `json:"properties,omitempty" yaml:"properties,omitempty"`
}

type ControllerConfigProperties struct {
	// key represents the variable
	For map[string]*ControllerConfigGvkObject `json:"for" yaml:"for"`
	// key represents the variable
	Own map[string]*ControllerConfigGvkObject `json:"own,omitempty" yaml:"own,omitempty"`
	// key represents the variable
	Watch map[string]*ControllerConfigGvkObject `json:"watch,omitempty" yaml:"watch,omitempty"`
	// key respresents the variable
	//Functions map[string]ControllerConfigFunctionBlock `json:",inline" yaml:",inline"`
	ControllerConfigPipelines []*ControllerConfigPipeline `json:"pipelines,omitempty" yaml:"pipelines,omitempty"`

	//Vars      []ControllerConfigVarBlock       `json:"vars,omitempty" yaml:"vars,omitempty"`
	//Functions []ControllerConfigFunctionsBlock `json:"fucntions,omitempty" yaml:"functions,omitempty"`
	//Services  []ControllerConfigFunctionsBlock `json:"services,omitempty" yaml:"services,omitempty"`
	//Services map[string]ControllerConfigFunction `json:"services,omitempty" yaml:"services,omitempty"`
}

type ControllerConfigGvkObject struct {
	//Gvr         *ControllerConfigGvr `json:"gvr" yaml:"gvr"`
	Resource    runtime.RawExtension `json:"resource,omitempty" yaml:"resource,omitempty"`
	PipelineRef string               `json:"pipelineRef,omitempty" yaml:"pipelineRef,omitempty"`
}

/*
	type ControllerConfigGvr struct {
		ApiVersion string `json:"apiVersion" yaml:"apiVersion"`
		Resource   string `json:"resource" yaml:"resource"`
	}
*/
type ControllerConfigPipeline struct {
	Name  string                               `json:"name" yaml:"name"`
	Vars  map[string]*ControllerConfigFunction `json:"vars,omitempty" yaml:"vars,omitempty"`
	Tasks map[string]*ControllerConfigFunction `json:"tasks,omitempty" yaml:"tasks,omitempty"`
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

type Type string

const (
	ForQueryType Type = "forQuery"
	QueryType    Type = "query"
	SliceType    Type = "slice"
	MapType      Type = "map"
	JQType       Type = "jq"
)

type ControllerConfigFunction struct {
	Block    `json:",inline" yaml:",inline"`
	Executor `json:",inline" yaml:",inline"`
	//ControllerConfigPipeline `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	Vars   map[string]*ControllerConfigFunction `json:"vars,omitempty" yaml:"vars,omitempty"`
	Type   Type                                 `json:"type,omitempty" yaml:"type,omitempty"`
	Config string                               `json:"config,omitempty" yaml:"config,omitempty"`
	// input is always a GVK of some sort
	Input *ControllerConfigInput `json:"input,omitempty" yaml:"input,omitempty"`
	// key = variableName, value is gvr format or not -> gvr format is needed for external resources
	Output map[string]*ControllerConfigOutput `json:"output,omitempty" yaml:"output,omitempty"`
}

type ControllerConfigOutput struct {
	//Gvr           *ControllerConfigGvr `json:"gvr,omitempty" yaml:"gvr,omitempty"`
	Resource      runtime.RawExtension `json:"resource,omitempty" yaml:"resource,omitempty"`
	GenericOutput map[string]string    `json:",inline" yaml:",inline"`
}

/*
type ControllerConfigGenericOutput struct {
	GenericInput string `json:",inline" yaml:",inline"`
}
*/

type ControllerConfigInput struct {
	//Gvr          *ControllerConfigGvr  `json:"gvr,omitempty" yaml:"gvr,omitempty"`
	Selector     *metav1.LabelSelector `json:"selector,omitempty" yaml:"selector,omitempty"`
	Key          string                `json:"key,omitempty" yaml:"key,omitempty"`
	Value        string                `json:"value,omitempty" yaml:"value,omitempty"`
	GenericInput map[string]string     `json:",inline" yaml:",inline"`
	Expression   string                `json:"expression,omitempty" yaml:"expression,omitempty"`
	Resource     runtime.RawExtension  `json:"resource,omitempty" yaml:"resource,omitempty"`
}

/*
type ControllerConfigGenericInput struct {
	GenericInput string `json:",inline" yaml:",inline"`
}
*/

type Executor struct {
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
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Items []ControllerConfig `json:"items" yaml:"items"`
}
