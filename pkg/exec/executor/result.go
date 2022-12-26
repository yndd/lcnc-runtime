package executor

/*
import (
	"fmt"
	"time"

	"github.com/yndd/lcnc-runtime/pkg/exec/output"
)

// ResultFunc is the callback used for gathering the
// result during graph execution.
type ResultFunc func(*result)

type result struct {
	vertexName string
	startTime  time.Time
	endTime    time.Time
	//outputCtx  map[string]*dag.OutputContext
	output  map[string]*output.OutputInfo
	success bool
	reason  string
}

func (r *exec) recordResult(re *result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.execResult = append(r.execResult, re)

	// update the final output
	for varName, outputInfo := range re.output {
		r.output.Update(varName, outputInfo)
	}

}

func (r *exec) PrintResult() {
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
*/
