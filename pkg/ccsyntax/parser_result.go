package ccsyntax

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Result struct {
	OriginContext *OriginContext `json:"inline" yaml:"inline"`
	Error         string         `json:"error,omitempty" yaml:"error,omitempty"`
}

type recordResultFn func(Result)

type OriginContext struct {
	//Index      int
	FOW             FOW                      `json:"fow,omitempty" yaml:"fow,omitempty"`
	RootVertexName  string                   `json:"rootVertexName,omitempty" yaml:"rootVertexName,omitempty"`
	GVK             *schema.GroupVersionKind `json:"gvk,omitempty" yaml:"gvk,omitempty"`
	Operation       Operation                `json:"operation,omitempty" yaml:"operation,omitempty"`
	Pipeline        string                   `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	Origin          Origin                   `json:"origin,omitempty" yaml:"origin,omitempty"`
	Block           bool                     `json:"block,omitempty" yaml:"block,omitempty"`
	BlockIndex      int                      `json:"blockIdx,omitempty" yaml:"blockIdx,omitempty"`
	BlockVertexName string                   `json:"blockVertexName,omitempty" yaml:"blockVertexName,omitempty"`
	VertexName      string                   `json:"vertexname,omitempty" yaml:"vertexname,omitempty"`
	LocalVarName    string                   `json:"localvarName,omitempty" yaml:"localvarName,omitempty"`
	LocalVars       map[string]string        `json:"localVars,omitempty" yaml:"localvarName,omitempty"`
}

func (in *OriginContext) DeepCopy() *OriginContext {
	if in == nil {
		return nil
	}
	out := new(OriginContext)
	in.DeepCopyInto(out)
	return out
}

func (in *OriginContext) DeepCopyInto(out *OriginContext) {
	*out = *in
}

type FOW string

const (
	FOWFor   FOW = "for"
	FOWOwn   FOW = "own"
	FOWWatch FOW = "watch"
)

type Operation string

const (
	OperationApply  Operation = "apply"
	OperationDelete Operation = "delete"
	OperationNone   Operation = "none"
)

type Origin string

const (
	OriginInvalid  Origin = "invalid"
	OriginFow      Origin = "fow"
	OriginVariable Origin = "vars"
	OriginFunction Origin = "function"
	OriginService  Origin = "services"
)
