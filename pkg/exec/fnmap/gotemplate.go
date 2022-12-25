package fnmap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"text/template"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
)

/*
func convert(i any) any {
	switch x := i.(type) {
	case map[any]any:
		nm := map[string]any{}
		for k, v := range x {
			nm[k.(string)] = convert(v)
		}
		return nm
	case map[string]any:
		for k, v := range x {
			x[k] = convert(v)
		}
	case []any:
		for k, v := range x {
			x[k] = convert(v)
		}
	}
	return i
}
*/

func (r *fnmap) runGT(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (map[string]*output.OutputInfo, error) {
	rx := &gt{
		outputContext: vertexContext.OutputContext,
	}
	/*
		in := convert(input)
		switch in := in.(type) {
		case map[string]any:
			input = in
		}
	*/
	fec := &fnExecConfig{
		executeRange:  true,
		executeSingle: true,
		// execution functions
		prepareInputFn: rx.prepareInput,
		runFn:          rx.runGT,
		// result functions
		initResultFn:   rx.initResult,
		recordResultFn: rx.recordResult,
		getResultFn:    rx.getResult,
	}

	return fec.run(ctx, vertexContext.Function, input)
}

type gt struct {
	m             sync.RWMutex
	result        []any
	outputContext map[string]*dag.OutputContext
}

func (r *gt) initResult(numItems int) {
	r.result = make([]any, 0, numItems)
}

func (r *gt) recordResult(o any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.result = append(r.result, o)
}

func (r *gt) getResult() map[string]*output.OutputInfo {
	res := make(map[string]*output.OutputInfo, 1)
	for varName, outputCtx := range r.outputContext {
		res[varName] = &output.OutputInfo{
			Internal: outputCtx.Internal,
			Value:    r.result,
		}
	}
	return res
}

func (r *gt) prepareInput(fnconfig *ctrlcfgv1.Function) any {
	if len(fnconfig.Input.Resource.Raw) != 0 {
		return string(fnconfig.Input.Resource.Raw)
	}
	return fnconfig.Input.Template
}

func (r *gt) runGT(ctx context.Context, extraInput any, input map[string]any) (any, error) {
	tmpl, ok := extraInput.(string)
	if !ok {
		return nil, fmt.Errorf("expecting string input in gotemplate, got: %T", extraInput)
	}

	if tmpl == "" {
		return nil, errors.New("missing template")
	}
	result := new(bytes.Buffer)
	// TODO: add template custom functions
	tpl, err := template.New("default").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return nil, err
	}
	fmt.Printf("runGT input: %v\n", input)
	err = tpl.Execute(result, input)
	if err != nil {
		return nil, err
	}
	var x any
	err = json.Unmarshal(result.Bytes(), &x)
	fmt.Printf("runGT result: %s", x)
	return x, err
}
