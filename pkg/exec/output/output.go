package output

import (
	"encoding/json"
	"fmt"

	"github.com/yndd/lcnc-runtime/pkg/ccutils/kv"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Output interface {
	kv.KV

	GetData(k string) any
	Print()
	GetFinalOutput() []any
}

type OutputInfo struct {
	Internal bool
	GVK      *schema.GroupVersionKind
	Data     any
}

func New() Output {
	return &output{
		o: kv.New(),
	}
}

type output struct {
	o kv.KV
}

func (r *output) AddEntry(k string, v any) {
	r.o.AddEntry(k, v)
}

func (r *output) Add(o kv.KV) {
	r.o.Add(o)
}

func (r *output) Get() map[string]any {
	return r.o.Get()
}

func (r *output) GetValue(k string) any {
	return r.o.GetValue(k)
}

func (r *output) Length() int {
	return r.o.Length()
}

func (r *output) GetData(k string) any {
	v := r.o.GetValue(k)
	oi, ok := v.(*OutputInfo)
	if !ok {
		return nil
	}
	return oi.Data
}

// used for debugging purposes
func (r *output) Print() {
	for varName, v := range r.o.Get() {
		oi, ok := v.(*OutputInfo)
		if !ok {
			fmt.Printf("unexpected outputInfo, got %T\n", v)
			continue
		}
		b, err := json.Marshal(oi.Data)
		if err != nil {
			fmt.Printf("output %s: marshal err %s\n", varName, err.Error())
		}
		fmt.Printf("  json output varName: %s internal: %t gvk: %v value:%s\n", varName, oi.Internal, oi.GVK, string(b))
	}
}

func (r *output) GetFinalOutput() []any {
	fo := []any{}
	for _, v := range r.o.Get() {
		oi, ok := v.(*OutputInfo)
		if !ok {
			fmt.Printf("unexpected outputInfo, got %T\n", v)
			continue
		}
		if !oi.Internal {
			fo = append(fo, oi.Data)
		}
	}
	return fo
}
