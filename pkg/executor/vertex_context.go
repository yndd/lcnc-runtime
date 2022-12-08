package executor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type vertexContext struct {
	vertexName string

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

	input map[string]*gojq.Code
	fn    string

	output map[string]interface{}
	status string
	reason string

	// callback
	recordResult ResultFunc
}

func (r *vertexContext) AddDoneCh(n string, c chan bool) {
	r.m.Lock()
	defer r.m.Unlock()
	r.doneChs[n] = c
}

func (r *vertexContext) AddDepCh(n string, c chan bool) {
	r.m.Lock()
	defer r.m.Unlock()
	r.depChs[n] = c
}

func (r *vertexContext) isFinished() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return !r.finished.IsZero()
}

/*
func (r *vertexcontext) hasStarted() bool {
	return !r.start.IsZero()
}
*/

func (r *vertexContext) isVisted() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return !r.visited.IsZero()
}

func (r *vertexContext) getDuration() time.Duration {
	return r.finished.Sub(r.start)
}

func (r *vertexContext) run(ctx context.Context) {

	r.m.Lock()
	r.start = time.Now()
	r.m.Unlock()
	// todo execute the function
	fmt.Printf("%s fn executed, doneChs: %v\n", r.vertexName, r.doneChs)
	r.m.Lock()
	r.finished = time.Now()
	r.m.Unlock()

	// callback function to capture the result
	r.recordResult(&ResultEntry{vertexName: r.vertexName, duration: r.getDuration()})

	// signal to the dependent function the result of the vertex fn execution
	for vertexName, doneCh := range r.doneChs {
		doneCh <- true
		close(doneCh)
		fmt.Printf("%s -> %s send done\n", r.vertexName, vertexName)
	}
	// signal the result of the vertex execution to the main walk
	r.doneFnCh <- true
	close(r.doneFnCh)
	fmt.Printf("%s -> walk main fn done\n", r.vertexName)
}

func (r *vertexContext) waitDependencies(ctx context.Context) bool {
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
