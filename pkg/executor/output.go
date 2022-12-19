package executor

import (
	"fmt"
	"strings"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

type Output interface {
	Update(vertexName string, o map[string]*ctrlcfgv1.Output, value any)
	Get(string) any
	GetFinalOutput() []any
	GetOutput()
}

func NewOutput() Output {
	return &output{
		o: map[string]*outputInfo{},
	}
}

type output struct {
	m sync.RWMutex
	o map[string]*outputInfo
}

type OutputKind string

const (
	variable OutputKind = "var"
	function OutputKind = "function"
)

type OutputResultKind string

const (
	final        OutputResultKind = "final"
	intermediate OutputResultKind = "intermediate"
)

type outputInfo struct {
	kind   OutputKind
	m      sync.RWMutex
	result map[string]*outputResult
}

type outputResult struct {
	kind  OutputResultKind
	value any
}

func (r *output) Update(vertexName string, oc map[string]*ctrlcfgv1.Output, value any) {
	r.m.Lock()
	defer r.m.Unlock()
	// initialize the context if the vertex is not yet initialized
	if _, ok := r.o[vertexName]; !ok {
		r.o[vertexName] = &outputInfo{
			result: map[string]*outputResult{},
		}
	}
	if oc == nil {
		// this is a variable -> the fn name = variable name
		r.o[vertexName].UpdateVar(vertexName, value)
	} else {
		for ocVarName, ocInfo := range oc {
			if len(ocInfo.Resource.Raw) != 0 {
				r.o[vertexName].UpdateFn(ocVarName, final, value)
			} else {
				r.o[vertexName].UpdateFn(ocVarName, intermediate, value)
			}
		}
	}
}

func (r *outputInfo) UpdateVar(vertexName string, value any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.kind = variable
	r.result[vertexName] = &outputResult{
		kind:  intermediate,
		value: value,
	}
}

func (r *outputInfo) UpdateFn(ocVarName string, k OutputResultKind, value any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.kind = function
	r.result[ocVarName] = &outputResult{
		kind:  k,
		value: value,
	}
}

func (r *output) Get(v string) any {
	r.m.RLock()
	defer r.m.RUnlock()

	s := strings.Split(v, ".")
	oi, ok := r.o[s[0]]
	if !ok {
		// not found -> should not happend
		return nil
	}
	oi.m.RLock()
	defer oi.m.RUnlock()
	if len(s) == 1 {
		return oi.result[s[0]].value
	}
	return oi.result[s[1]].value
}

func (r *output) GetFinalOutput() []any {
	r.m.RLock()
	defer r.m.RUnlock()

	fo := []any{}
	for _, oi := range r.o {
		oi.m.RLock()
		defer oi.m.RUnlock()
		for _, or := range oi.result {
			if or.kind == final {
				fo = append(fo, or.value)
			}
		}
	}
	return fo
}

// used for debugging purposes
func (r *output) GetOutput() {
	r.m.RLock()
	defer r.m.RUnlock()

	for vertexName, oi := range r.o {
		oi.m.RLock()
		defer oi.m.RUnlock()
		for outputName, or := range oi.result {
			var varName string
			if outputName == vertexName {
				varName = vertexName
			} else {
				varName = fmt.Sprintf("%s.%s", vertexName, outputName)
			}
			fmt.Printf("output varName: %s, type: %s kind: %s value: %#v\n", varName, oi.kind, or.kind, or.value)
		}
	}
}

func (r *executor) GetOutput() {
	r.output.GetOutput()
}
