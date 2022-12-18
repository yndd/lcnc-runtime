package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/fnmap"
	ctrl "sigs.k8s.io/controller-runtime"
)

type execContext struct {
	vertexName string
	fnMap      fnmap.FnMap

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
	recordResult ResultFunc
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

/*
func (r *vertexcontext) hasStarted() bool {
	return !r.start.IsZero()
}
*/

func (r *execContext) isVisted() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return !r.visited.IsZero()
}

/*
func (r *vertexContext) getDuration() time.Duration {
	return r.finished.Sub(r.start)
}
*/

func (r *execContext) run(ctx context.Context, req ctrl.Request) {
	r.m.Lock()
	r.start = time.Now()
	r.m.Unlock()

	var success bool
	var reason string
	// todo execute the function
	switch r.vertexContext.Function.Type {
	case ctrlcfgv1.Container, ctrlcfgv1.Wasm:
	case ctrlcfgv1.ForQueryType:
		// we use a dedicated key for the for
		x, err := r.fnMap.RunFn(ctx, r.vertexContext.Function, map[string]any{fnmap.ForKey: req.NamespacedName})
		if err != nil {
			success = false
			reason = err.Error()
			fmt.Printf("vertex: %s, output: %v, err: %s\n", r.vertexName, x, err.Error())
		} else {
			success = true
			fmt.Printf("vertex: %s, output: %v\n", r.vertexName, x)
		}

	case ctrlcfgv1.QueryType:
		x, err := r.fnMap.RunFn(ctx, r.vertexContext.Function, nil)
		if err != nil {
			success = false
			reason = err.Error()
			fmt.Printf("vertex: %s, output: %v, err: %s\n", r.vertexName, x, err.Error())
		} else {
			success = true
			fmt.Printf("vertex: %s, output: %v\n", r.vertexName, x)
		}
	case ctrlcfgv1.MapType, ctrlcfgv1.SliceType, ctrlcfgv1.JQType:
	}

	fmt.Printf("%s fn executed, doneChs: %v\n", r.vertexName, r.doneChs)
	r.m.Lock()
	r.finished = time.Now()
	r.m.Unlock()

	// callback function to capture the result
	r.recordResult(&result{
		vertexName: r.vertexName,
		startTime:  r.start,
		endTime:    r.finished,
		success:    success,
		reason:     reason,
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
