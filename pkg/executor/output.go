package executor

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/yndd/lcnc-runtime/pkg/fnmap"
)

type Output interface {
	Update(vertexName, varName string, oc *fnmap.Output)
	Get(string) any
	GetFinalOutput() []any
	GetOutput()
}

func NewOutput() Output {
	return &output{
		o: map[string]*outputInfo{},
	}
}

type OutputKind string

type output struct {
	m sync.RWMutex
	o map[string]*outputInfo
}

type outputInfo struct {
	m      sync.RWMutex
	result map[string]*outputResult
}

type outputResult struct {
	internal bool
	value    any
}

func (r *output) Update(vertexName, varName string, oc *fnmap.Output) {
	r.m.Lock()
	defer r.m.Unlock()
	// initialize the context if the vertex is not yet initialized
	if _, ok := r.o[vertexName]; !ok {
		r.o[vertexName] = &outputInfo{
			result: map[string]*outputResult{},
		}
	}
	r.o[vertexName].Update(varName, oc)

}

func (r *outputInfo) Update(varName string, oc *fnmap.Output) {
	r.m.Lock()
	defer r.m.Unlock()
	r.result[varName] = &outputResult{
		internal: oc.Internal,
		value:    oc.Value,
	}
}

func (r *output) Get(v string) any {
	r.m.RLock()
	defer r.m.RUnlock()

	s := strings.Split(v, ".")
	oi, ok := r.o[s[0]]
	fmt.Printf("output get ref: %s, oi: %v, ok: %t\n", v, oi, ok)
	if !ok {
		// not found -> should not happend
		return nil
	}
	oi.m.RLock()
	defer oi.m.RUnlock()
	if len(s) == 1 {
		fmt.Printf("output get ref: %s, oi result: %v, ok: %t\n", v, oi.result[s[0]].value, ok)
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
			if !or.internal {
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
			b, err := json.Marshal(or.value)
			if err != nil {
				fmt.Printf("output %s: marshal err %s\n", varName, err.Error())
			}
			fmt.Printf("output %s: json %s\n", varName, string(b))
			//fmt.Printf("output %s value: %#v\n", varName, or.value)
			//fmt.Printf("output varName: %s, type: %s kind: %s value: %#v\n", varName, oi.kind, or.kind, or.value)
		}
	}
}

func (r *executor) GetOutput() {
	r.output.GetOutput()
}
