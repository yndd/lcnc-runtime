package executor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type execContext struct {
	execName   string
	vertexName string

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
	// identifies the time the vertex fn finished
	finished time.Time

	//vertexContext *rtdag.VertexContext
	vertexContext any

	// handler
	vertexFuntionRunFn VertexFuntionRunFn
}

type VertexResult struct {
	Start   time.Time
	Finish  time.Time
	Success bool
	Reason  string
	Input   any
	Output  any
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
	// execute the handler that runs the function
	success := r.vertexFuntionRunFn(ctx, r.vertexName, r.vertexContext)
	r.finished = time.Now()
	// signal to the dependent function the result of the vertex fn execution
	for vertexName, doneCh := range r.doneChs {
		doneCh <- success
		close(doneCh)
		fmt.Printf("execContext execName %s vertexName: %s -> %s send done\n", r.execName, r.vertexName, vertexName)
	}
	// signal the result of the vertex execution to the main walk
	r.doneFnCh <- success
	close(r.doneFnCh)
	fmt.Printf("execContext execName %s vertexName: %s -> walk main fn done\n", r.execName, r.vertexName)
}

func (r *execContext) waitDependencies(ctx context.Context) bool {
	// for each dependency wait till a it completed, either through
	// the dependency Channel or cancel or

	fmt.Printf("execContext execName %s vertexName: %s wait dependencies: %v\n", r.execName, r.vertexName, r.depChs)
DepSatisfied:
	for depVertexName, depCh := range r.depChs {
		//fmt.Printf("waitDependencies %s -> %s\n", depVertexName, r.vertexName)
		//DepSatisfied:
		for {
			select {
			case d, ok := <-depCh:
				fmt.Printf("execContext execName %s: %s -> %s rcvd done, d: %t, ok: %t\n", r.execName, depVertexName, r.vertexName, d, ok)
				if ok {
					continue DepSatisfied
				}
				if !d {
					// dependency failed
					return false
				}
				continue DepSatisfied
			case <-time.After(time.Second * 5):
				fmt.Printf("execContext execName %s vertexName: %s wait timeout, is waiting for %s\n", r.execName, r.vertexName, depVertexName)
			}
		}
	}
	fmt.Printf("execContext execName %s vertexName: %s finished waiting\n", r.execName, r.vertexName)
	return true
}
