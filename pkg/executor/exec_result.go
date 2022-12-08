package executor

import (
	"fmt"
	"time"
)

// ResultFunc is the callback used for gathering the
// result during graph execution.
type ResultFunc func(*ResultEntry)

type ResultEntry struct {
	vertexName string
	duration   time.Duration
	ouput      map[string]interface{}
	status     string
	reason     string
}

func (r *exectutor) recordResult(re *ResultEntry) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, re)
}

func (r *exectutor) GetWalkResult() {
	r.mr.RLock()
	defer r.mr.RUnlock()
	for i, result := range r.result {
		fmt.Printf("result order: %d vertex: %s, duration %s\n", i, result.vertexName, result.duration)
	}
}
