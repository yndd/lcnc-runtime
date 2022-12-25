package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
)

type execContext struct {
	vertexName     string
	rootVertexName string
	fnMap          fnmap.FnMap

	// used to handle the dependencies between the functions
	m sync.RWMutex
	// used to send fn result from the src function
	// to the dependent function
	doneChs map[string]chan bool
	// used by the dependent vertex function to rcv the result
	// of the dependent src function
	depChs map[string]chan bool

	// used to signal the vertex function is done
	// to the main walk entry
	doneFnCh chan bool

	// identifies the time the vertex got scheduled
	visited time.Time
	// identifies the time the vertex fn started
	start time.Time
	// identifies the time the vertex fn finished
	finished time.Time

	vertexContext *dag.VertexContext

	// callback
	recordResult result.RecordResultFn
	recordOutput output.RecordOutputFn
	// output
	output output.Output
}

func (r *execContext) AddDoneCh(n string, c chan bool) {
	r.m.Lock()
	defer r.m.Unlock()
	r.doneChs[n] = c
}

func (r *execContext) AddDepCh(n string, c chan bool) {
	r.m.Lock()
	defer r.m.Unlock()
	r.depChs[n] = c
}

func (r *execContext) isFinished() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return !r.finished.IsZero()
}

func (r *execContext) isVisted() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return !r.visited.IsZero()
}

func (r *execContext) run(ctx context.Context) {
	r.m.Lock()
	r.start = time.Now()
	r.m.Unlock()

	// Gather the input based on the function type
	// Determine if this is an internal fn runner or not
	input := map[string]any{}
	switch r.vertexContext.Function.Type {
	case ctrlcfgv1.ContainerType, ctrlcfgv1.WasmType:
		input[r.rootVertexName] = r.output.Get(r.rootVertexName)
		for _, ref := range r.vertexContext.References {
			input[ref] = r.output.Get(ref)
		}
	case ctrlcfgv1.ForInitType:
		input[r.vertexName] = r.output.Get(r.vertexName)
	default:
		fmt.Printf("references: %v\n", r.vertexContext.References)
		for _, ref := range r.vertexContext.References {
			input[ref] = r.output.Get(ref)
		}
	}
	fmt.Printf("vertex: %s input: %#v\n", r.vertexName, input)

	// Run the execution context
	success := true
	reason := ""
	o, err := r.fnMap.RunFn(ctx, r.vertexContext, input)
	if err != nil {
		if !errors.Is(err, fnmap.ErrConditionFalse) {
			success = false
		}
		reason = err.Error()
	}

	for varname, oc := range o {
		fmt.Printf("vertex: %s varname: %s, success: %t, reason: %s, output: %v\n", r.vertexName, varname, success, reason, oc.Value)
	}

	fmt.Printf("%s fn executed, doneChs: %v\n", r.vertexName, r.doneChs)
	r.m.Lock()
	r.finished = time.Now()
	r.m.Unlock()

	for varName, outputInfo := range o {
		r.recordOutput(varName, outputInfo)
	}

	// callback function to capture the result
	r.recordResult(&result.ResultInfo{
		VertexName: r.vertexName,
		StartTime:  r.start,
		EndTime:    r.finished,
		Input:      input,
		Output:     o,
		Success:    success,
		Reason:     reason,
	})

	// signal to the dependent function the result of the vertex fn execution
	for vertexName, doneCh := range r.doneChs {
		doneCh <- success
		close(doneCh)
		fmt.Printf("%s -> %s send done\n", r.vertexName, vertexName)
	}
	// signal the result of the vertex execution to the main walk
	r.doneFnCh <- success
	close(r.doneFnCh)
	fmt.Printf("%s -> walk main fn done\n", r.vertexName)
}

func (r *execContext) waitDependencies(ctx context.Context) bool {
	// for each dependency wait till a it completed, either through
	// the dependency Channel or cancel or

	fmt.Printf("%s wait dependencies: %v\n", r.vertexName, r.depChs)
DepSatisfied:
	for depVertexName, depCh := range r.depChs {
		//fmt.Printf("waitDependencies %s -> %s\n", depVertexName, r.vertexName)
		//DepSatisfied:
		for {
			select {
			case d, ok := <-depCh:
				fmt.Printf("%s -> %s rcvd done, d: %t, ok: %t\n", depVertexName, r.vertexName, d, ok)
				if ok {
					continue DepSatisfied
				}
				if !d {
					// dependency failed
					return false
				}
				continue DepSatisfied
			case <-time.After(time.Second * 5):
				fmt.Printf("wait timeout vertex: %s is waiting for %s\n", r.vertexName, depVertexName)
			}
		}
	}
	fmt.Printf("%s finished waiting\n", r.vertexName)
	return true
}
