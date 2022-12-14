package scheduler

import (
	"fmt"
	"time"

	"github.com/yndd/lcnc-runtime/pkg/dag"
)

// ResultFunc is the callback used for gathering the
// result during graph execution.
type ResultFunc func(*result)

type result struct {
	vertexName    string
	startTime     time.Time
	endTime       time.Time
	vertexContext *dag.VertexContext
	status        string
	reason        string
}

func (r *scheduler) recordResult(re *result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.execResult = append(r.execResult, re)
}

func (r *scheduler) GetWalkResult() {
	r.mr.RLock()
	defer r.mr.RUnlock()
	for i, result := range r.execResult {
		fmt.Printf("result order: %d vertex: %s, duration %s\n", i, result.vertexName, result.endTime.Sub(result.startTime))
	}
}
