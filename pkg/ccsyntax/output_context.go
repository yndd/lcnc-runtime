package ccsyntax

import (
	"fmt"
	"sync"

	"github.com/yndd/lcnc-runtime/pkg/dag"
)

// OutputContext stores the output context in a global DAG for validating
// that the outputs are globally unique. This is only used by the parser
// for resolving and connecting the runtime graph
// We also store the output conetxt in the runtime DAG for runtime operation
type OutputContext interface {
	GetName() string
	Add(fe FOWEntry)
	GetDAG(fe FOWEntry) dag.DAG
	Print()
}

type outputContext struct {
	name string
	m    sync.RWMutex
	o    map[FOWEntry]dag.DAG
}

type FOWEntry struct {
	FOW            FOW
	RootVertexName string
}

func NewOutputContext(n string) OutputContext {
	return &outputContext{
		name: n,
		o:    make(map[FOWEntry]dag.DAG),
	}
}

func (r *outputContext) GetName() string {
	return r.name
}

func (r *outputContext) Add(fe FOWEntry) {
	r.m.Lock()
	defer r.m.Unlock()
	if _, ok := r.o[fe]; !ok {
		r.o[fe] = dag.New()
	}
}

func (r *outputContext) GetDAG(fe FOWEntry) dag.DAG {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.o[fe]
}

func (r *outputContext) Print() {
	fmt.Printf("Name: %s\n", r.name)
	for fe, d := range r.o {
		fmt.Printf("FOW: %s, RootVertexname: %s\n", fe.FOW, fe.RootVertexName)
		d.PrintVertices()
	}
}
