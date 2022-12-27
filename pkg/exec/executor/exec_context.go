package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/exec/input"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
)

type execContext struct {
	vertexName string
	cfg        *Config

	// used to signal the vertex function is done
	// to the main walk entry
	doneFnCh chan bool
	// used to handle the dependencies between the functions
	m sync.RWMutex
	// used to send fn result from the src function
	// to the dependent function
	doneChs map[string]chan bool
	// used by the dependent vertex function to rcv the result
	// of the dependent src function
	depChs map[string]chan bool
	// identifies the time the vertex got scheduled
	visited time.Time
	// identifies the time the vertex fn started
	start time.Time
	// identifies the time the vertex fn finished
	finished time.Time

	vertexContext *rtdag.VertexContext

	// callback
	recordResult result.RecordResultFn
	recordOutput output.RecordOutputFn
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
	fmt.Printf("exec isVisited: %#v\n", r)
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
	//input := map[string]any{}
	i := input.New()
	switch r.vertexContext.Function.Type {
	case ctrlcfgv1.RootType:
		// this is a dummy function, input is not relevant
	case ctrlcfgv1.ContainerType, ctrlcfgv1.WasmType:
		i.Add(r.cfg.RootVertexName, r.cfg.Output.GetValue(r.cfg.RootVertexName))
		//input[r.cfg.RootVertexName] = r.cfg.Output.GetValue(r.cfg.RootVertexName)
		for _, ref := range r.vertexContext.References {
			//input[ref] = r.cfg.Output.GetValue(ref)
			i.Add(ref, r.cfg.Output.GetValue(ref))
		}

	default:
		fmt.Printf("execContext execName %s vertexName: %s references: %v\n", r.cfg.Name, r.vertexName, r.vertexContext.References)
		for _, ref := range r.vertexContext.References {
			//input[ref] = r.cfg.Output.GetValue(ref)
			i.Add(ref, r.cfg.Output.GetValue(ref))
		}
	}
	fmt.Printf("execContext execName %s vertexName: %s input: %#v\n", r.cfg.Name, r.vertexName, i.Get())

	// Run the execution context

	success := true
	reason := ""
	o, err := r.cfg.FnMap.Run(ctx, r.vertexContext, i)
	if err != nil {
		if !errors.Is(err, ErrConditionFalse) {
			success = false
		}
		reason = err.Error()
	}
	fmt.Printf("execContext execName %s vertexName: %s fn executed, doneChs: %v\n", r.cfg.Name, r.vertexName, r.doneChs)
	r.m.Lock()
	r.finished = time.Now()
	r.m.Unlock()

	for varName, outputInfo := range o.GetOutputInfo() {
		fmt.Printf("execContext execName: %s, vertexName: %s output  varname: %s, success: %t, reason: %s, output: %v\n", r.cfg.Name, r.vertexName, varName, success, reason, outputInfo.Value)
		r.recordOutput(varName, outputInfo)
	}

	// callback function to capture the result
	r.recordResult(&result.ResultInfo{
		Type:       r.cfg.Type,
		ExecName:   r.cfg.Name,
		VertexName: r.vertexName,
		StartTime:  r.start,
		EndTime:    r.finished,
		Input:      i,
		Output:     o,
		Success:    success,
		Reason:     reason,
	})

	// signal to the dependent function the result of the vertex fn execution
	for vertexName, doneCh := range r.doneChs {
		doneCh <- success
		close(doneCh)
		fmt.Printf("execContext execName %s vertexName: %s -> %s send done\n", r.cfg.Name, r.vertexName, vertexName)
	}
	// signal the result of the vertex execution to the main walk
	r.doneFnCh <- success
	close(r.doneFnCh)
	fmt.Printf("execContext execName %s vertexName: %s -> walk main fn done\n", r.cfg.Name, r.vertexName)
}

func (r *execContext) waitDependencies(ctx context.Context) bool {
	// for each dependency wait till a it completed, either through
	// the dependency Channel or cancel or

	fmt.Printf("execContext execName %s vertexName: %s wait dependencies: %v\n", r.cfg.Name, r.vertexName, r.depChs)
DepSatisfied:
	for depVertexName, depCh := range r.depChs {
		//fmt.Printf("waitDependencies %s -> %s\n", depVertexName, r.vertexName)
		//DepSatisfied:
		for {
			select {
			case d, ok := <-depCh:
				fmt.Printf("execContext execName %s: %s -> %s rcvd done, d: %t, ok: %t\n", r.cfg.Name, depVertexName, r.vertexName, d, ok)
				if ok {
					continue DepSatisfied
				}
				if !d {
					// dependency failed
					return false
				}
				continue DepSatisfied
			case <-time.After(time.Second * 5):
				fmt.Printf("execContext execName %s vertexName: %s wait timeout, is waiting for %s\n", r.cfg.Name, r.vertexName, depVertexName)
			}
		}
	}
	fmt.Printf("execContext execName %s vertexName: %s finished waiting\n", r.cfg.Name, r.vertexName)
	return true
}
