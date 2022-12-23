package executor

import (
	"fmt"
	"time"

	"github.com/yndd/lcnc-runtime/pkg/fnmap"
)

// ResultFunc is the callback used for gathering the
// result during graph execution.
type ResultFunc func(*result)

type result struct {
	vertexName string
	startTime  time.Time
	endTime    time.Time
	//outputCtx  map[string]*dag.OutputContext
	output  map[string]*fnmap.Output
	success bool
	reason  string
}

func (r *executor) recordResult(re *result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.execResult = append(r.execResult, re)

	// there should only be 1 output so this is safe
	for varName, outputCtx := range re.output {
		r.output.Update(re.vertexName, varName, outputCtx)
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
