package result

import (
	"fmt"
	"sync"
	"time"

	"github.com/yndd/lcnc-runtime/pkg/exec/output"
)

type Result interface {
	RecordResult(ri *ResultInfo)
	GetResult() []*ResultInfo
	PrintResult()
}

type RecordResultFn func(*ResultInfo)

type ExecType string

const (
	ExecRootType  ExecType = "root"
	ExecBlockType ExecType = "block"
)

type ResultInfo struct {
	Type        ExecType
	ExecName    string
	VertexName  string
	StartTime   time.Time
	EndTime     time.Time
	Input       map[string]any
	Output      output.Output
	Success     bool
	Reason      string
	BlockResult Result
}

func New() Result {
	return &result{
		r: make([]*ResultInfo, 0),
	}
}

type result struct {
	m sync.RWMutex
	r []*ResultInfo
}

func (r *result) RecordResult(ri *ResultInfo) {
	r.m.Lock()
	defer r.m.Unlock()
	r.r = append(r.r, ri)
}

func (r *result) GetResult() []*ResultInfo {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.r
}

func (r *result) PrintResult() {
	r.m.RLock()
	defer r.m.RUnlock()
	totalSuccess := true
	var totalDuration time.Duration
	for i, ri := range r.r {
		if ri.Type == ExecRootType && ri.VertexName == "total" {
			totalDuration = ri.EndTime.Sub(ri.StartTime)
		} else {
			s := "OK"
			if !ri.Success {
				totalSuccess = false
				s = "NOK"
			}
			fmt.Printf("  result order: %d exec: %s vertex: %s, duration %s, success: %s, reason: %s\n",
				i,
				ri.ExecName,
				ri.VertexName,
				ri.EndTime.Sub(ri.StartTime),
				s,
				ri.Reason,
			)

			if ri.BlockResult != nil {
				ri.BlockResult.PrintResult()
			}
		}
	}
	s := "OK"
	if !totalSuccess {
		s = "NOK"
	}
	fmt.Printf("overall result duration: %s, success: %s\n", totalDuration, s)
}
