package input

import (
	"encoding/json"
	"fmt"

	"github.com/yndd/lcnc-runtime/pkg/kv"
)

type Input interface {
	kv.KV

	Print(s string)
}

func New() Input {
	return &input{
		i: kv.New(),
	}
}

type input struct {
	i kv.KV
}

func (r *input) AddEntry(k string, v any) {
	r.i.AddEntry(k, v)
}

func (r *input) Add(i kv.KV) {
	r.i.Add(i)
}

func (r *input) Get() map[string]any {
	return r.i.Get()
}

func (r *input) GetValue(k string) any {
	return r.i.GetValue(k)
}

func (r *input) Length() int {
	return r.i.Length()
}

func (r *input) Print(vertexName string) {
	for varName, v := range r.i.Get() {
		b, err := json.Marshal(v)
		if err != nil {
			fmt.Printf("input vertexName: %s, varName: %s: marshal err %s\n", vertexName, varName, err.Error())
		}
		fmt.Printf("json input vertexName: %s, varName: %s value:%s\n", vertexName, varName, string(b))
	}
}
