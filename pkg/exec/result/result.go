package result

import (
	"fmt"
	"time"

	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/ccutils/slice"
)

type Result interface {
	slice.Slice
	Print()
}

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
	Input       input.Input
	Output      output.Output
	Success     bool
	Reason      string
	BlockResult Result
}

func New() Result {
	return &result{
		r: slice.New(),
	}
}

type result struct {
	r slice.Slice
}

func (r *result) Add(v any) {
	r.r.Add(v)
}

func (r *result) Get() []any {
	return r.r.Get()
}

func (r *result) Length() int {
	return r.r.Length()
}

func (r *result) Print() {
	totalSuccess := true
	var totalDuration time.Duration
	for i, v := range r.r.Get() {
		ri, ok := v.(*ResultInfo)
		if !ok {
			fmt.Printf("unexpected resultInfo, got %T\n", v)
		}
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
				ri.BlockResult.Print()
			}
		}
	}
	s := "OK"
	if !totalSuccess {
		s = "NOK"
	}
	fmt.Printf("overall result duration: %s, success: %s\n", totalDuration, s)
}
