package executor

import (
	"fmt"
	"time"

	"github.com/yndd/lcnc-runtime/pkg/dag"
)

// ResultFunc is the callback used for gathering the
// result during graph execution.
type ResultFunc func(*execResult)

type execResult struct {
	vertexName    string
	startTime     time.Time
	endTime       time.Time
	vertexContext *dag.VertexContext
	status        string
	reason        string
}

func (r *exectutor) recordResult(re *execResult) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.execResult = append(r.execResult, re)
}

func (r *exectutor) GetWalkResult() {
	r.mr.RLock()
	defer r.mr.RUnlock()
	for i, result := range r.execResult {
		fmt.Printf("result order: %d vertex: %s, duration %s\n", i, result.vertexName, result.endTime.Sub(result.startTime))
	}
}
