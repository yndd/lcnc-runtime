package output

import (
	"encoding/json"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Output interface {
	RecordOutput(varName string, oi *OutputInfo)
	GetOutputInfo() map[string]*OutputInfo
	GetValue(string) any
	GetFinalOutput() []any
	PrintOutput()
}

type RecordOutputFn func(varName string, oi *OutputInfo)

type OutputInfo struct {
	Internal bool
	GVK      *schema.GroupVersionKind
	Value    any
}

func New() Output {
	return &output{
		o: map[string]*OutputInfo{},
	}
}

type output struct {
	m sync.RWMutex
	o map[string]*OutputInfo
}

func (r *output) RecordOutput(varName string, oi *OutputInfo) {
	r.m.Lock()
	defer r.m.Unlock()

	r.o[varName] = oi
}

func (r *output) GetOutputInfo() map[string]*OutputInfo {
	o := make(map[string]*OutputInfo, len(r.o))
	for k, v := range r.o {
		o[k] = v
	}
	return o
}

func (r *output) GetValue(v string) any {
	r.m.RLock()
	defer r.m.RUnlock()

	oi, ok := r.o[v]
	if !ok {
		// not found -> should not happen
		return nil
	}
	return oi.Value
}

func (r *output) GetFinalOutput() []any {
	r.m.RLock()
	defer r.m.RUnlock()

	fo := []any{}
	for _, oi := range r.o {
		if !oi.Internal {
			fo = append(fo, oi.Value)
		}
	}
	return fo
}

// used for debugging purposes
func (r *output) PrintOutput() {
	r.m.RLock()
	defer r.m.RUnlock()

	for varName, oi := range r.o {
		b, err := json.Marshal(oi.Value)
		if err != nil {
			fmt.Printf("output %s: marshal err %s\n", varName, err.Error())
		}
		fmt.Printf("  json output varName: %s internal: %t gvk: %v value:%s\n", varName, oi.Internal, oi.GVK, string(b))
	}
}
