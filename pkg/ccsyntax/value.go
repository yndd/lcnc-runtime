package ccsyntax

import (
	"fmt"
	"strings"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ValueKind string

const (
	VariableKind               ValueKind = "variable"
	KeyVariableReferenceKind   ValueKind = "keyVariableReference" // used for $KEY, $VALUE
	RootVariableReferenceKind  ValueKind = "rootVariableReference"
	ChildVariableReferenceKind ValueKind = "childVariableRefrence"
	GVRKind                    ValueKind = "gvr"
)

type Value struct {
	Kind     ValueKind
	Variable []string
	Gvr      *schema.GroupVersionResource
}

func GetValue(s string) (*Value, error) {
	if len(s) <= 1 {
		return nil, fmt.Errorf("input value should have an input string with len > 1, got: %s", s)
	}
	// check if this is a variable or a gvr
	if string(s[0:1]) == "$" {
		varKind, varValue := GetVariable(s)
		return &Value{
			Kind:     varKind,
			Variable: varValue,
		}, nil
	}
	if len(strings.Split(s, "/")) == 1 {
		// this is a regular variable w/o a reference ($)
		return &Value{
			Kind:     VariableKind,
			Variable: []string{s},
		}, nil
	}
	// this is a gvr
	gvr, err := ctrlcfgv1.GetGVR(s)
	if err != nil {
		return nil, err
	}
	return &Value{
		Kind: GVRKind,
		Gvr:  gvr,
	}, nil

}

func GetVariable(s string) (ValueKind, []string) {
	// remove the first char from the string
	// split the string with the . delineator
	split := strings.Split(s[1:], ".")
	if split[0] == "VALUE" || split[0] == "KEY" {
		return KeyVariableReferenceKind, split
	}
	if len(split) > 1 {
		return ChildVariableReferenceKind, split
	}
	return RootVariableReferenceKind, split
}
