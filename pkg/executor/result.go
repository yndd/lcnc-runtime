package executor

import (
	"encoding/json"
	"fmt"
	"time"

	rctxv1 "github.com/yndd/lcnc-runtime/pkg/api/resourcecontext/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/meta"
)

// ResultFunc is the callback used for gathering the
// result during graph execution.
type ResultFunc func(*result)

type result struct {
	vertexName string
	startTime  time.Time
	endTime    time.Time
	outputCtx  map[string]*dag.OutputContext
	output     any
	success    bool
	reason     string
}

func (r *executor) recordResult(re *result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.execResult = append(r.execResult, re)

	// we can have 2 outcomes a regular output from an internal function
	switch o := re.output.(type) {
	case *rctxv1.ResourceContext:
		// range over the output
		for gvkString, rctxOutput := range o.Spec.Properties.Output {
			for varName, outputCtx := range re.outputCtx {
				if gvkString == meta.GVKToString(outputCtx.GVK) {
					output := make([]any, 0, len(rctxOutput))
					for _, rctxo := range rctxOutput {
						x := map[string]any{}
						if err := json.Unmarshal([]byte(rctxo), &x); err != nil {
							fmt.Printf("error unmarshaling the data, err: %s\n", err.Error())
						}
						output = append(output, x)
					}
					r.output.Update(re.vertexName, varName, outputCtx, output)
				}
			}
		}
	default:
		// there should only be 1 output so this is safe
		for varName, outputCtx := range re.outputCtx {
			r.output.Update(re.vertexName, varName, outputCtx, re.output)
		}
	}
}

func (r *executor) GetResult() {
	r.mr.RLock()
	defer r.mr.RUnlock()
	overallSuccess := true
	var overallDuration time.Duration
	for i, result := range r.execResult {
		if result.vertexName == "total" {
			overallDuration = result.endTime.Sub(result.startTime)
		} else {
			fmt.Printf("result order: %d vertex: %s, duration %s, success: %t, reason: %s\n",
				i,
				result.vertexName,
				result.endTime.Sub(result.startTime),
				result.success,
				result.reason)

			if !result.success {
				overallSuccess = false
			}
		}
	}
	fmt.Printf("overall result success : %t, duration: %s\n", overallSuccess, overallDuration)
}
