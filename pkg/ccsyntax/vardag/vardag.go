package vardag

import (
	"fmt"

	"github.com/yndd/lcnc-runtime/pkg/dag"
)

type VarDAG interface {
	AddVariable(s string, v *VariableContext) error
	VarExists(s string) bool
	GetVarInfo(s string) *VariableContext
	//GetVertices() map[string]*OutputContext
	//GetReferenceInfo(s string) (*OutputContext, error)
	Print()
}

func New() VarDAG {
	return &varDAG{
		d: dag.New(),
	}
}

type varDAG struct {
	d dag.DAG
}

type VariableContext struct {
	VertexName      string // name of the vertex
	OutputVertex    string // used for validation
	BlockIndex      int    // used for validation and connectivity
	BlockVertexName string // used for validation and connectivity
}

func (r *varDAG) AddVariable(s string, v *VariableContext) error {
	fmt.Printf("addVariable: %s, variableContext: %v\n", s, *v)
	return r.d.AddVertex(s, v)
}

func (r *varDAG) VarExists(s string) bool {
	return r.d.VertexExists(s)
}

func (r *varDAG) GetVarInfo(s string) *VariableContext {
	v := r.d.GetVertex(s)
	fmt.Printf("getVarInfo: %s, variableContext: %#v\n", s, v)
	oc, ok := v.(*VariableContext)
	if ok {
		return oc
	}
	return nil
}
func (r *varDAG) getVariables() map[string]*VariableContext {
	vs := r.d.GetVertices()
	ocs := map[string]*VariableContext{}
	for vertexName, v := range vs {
		oc, ok := v.(*VariableContext)
		if ok {
			ocs[vertexName] = oc
		}
	}
	return ocs
}

func (r *varDAG) Print() {
	fmt.Printf("###### VAR DAG start #######\n")
	for varName, vc := range r.getVariables() {
		fmt.Printf("varName: %s varContext: %v\n", varName, *vc)
	}
	fmt.Printf("###### VAR DAG stop #######\n")
}
