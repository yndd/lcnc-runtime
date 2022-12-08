package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/henderiw-k8s-lcnc/lcnc-runtime2/pkg/dag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type exectutor struct {
	cancelFn context.CancelFunc

	// used during the Walk func
	mw        sync.RWMutex
	walkMap   map[string]*vertexContext
	fnDoneMap map[string]chan bool

	mr     sync.RWMutex
	execResult []*execResult
	finaloutput map[string][]unstructured.Unstructured

}

type execResult struct {
	vertexName string
	startTime  string
	endTime    string
	input      map[string]interface{}
	output     map[string]interface{}
	status     string
	reason     string
}

func New() *exectutor {
	return &exectutor{
		mw:        sync.RWMutex{},
		walkMap:   map[string]*vertexContext{},
		fnDoneMap: map[string]chan bool{},
		mr:        sync.RWMutex{},
		result:    []*ResultEntry{},
	}
}

func (r *exectutor) Walk(ctx context.Context, d dag.DAG, from string) {
	// walk initialization
	r.initWalk(d)
	start := time.Now()
	ctx, cancelFn := context.WithCancel(ctx)
	// to be changed
	r.cancelFn = cancelFn
	r.walk(ctx, d, from, true, 1)
	// add total as a last entry in the result
	r.recordResult(&ResultEntry{
		vertexName: "total",
		duration:   time.Since(start),
	})
}

func (r *exectutor) initWalk(d dag.DAG) {
	//d.wg = new(sync.WaitGroup)
	r.result = []*ResultEntry{}
	r.walkMap = map[string]*vertexContext{}
	for vertexName := range d.GetVertices() {
		//fmt.Printf("init vertexName: %s\n", vertexName)
		r.walkMap[vertexName] = &vertexContext{
			vertexName: vertexName,
			//cancelFn:   cancelFn,
			doneChs: make(map[string]chan bool), //snd
			depChs:  make(map[string]chan bool), //rcv
			// callback to gather the result
			recordResult: r.recordResult,
		}
	}
	// build the channel matrix to signal dependencies through channels
	// for every dependency (upstream relationship between verteces)
	// create a channel
	// add the channel to the upstreamm vertex doneCh map ->
	// usedto signal/send the vertex finished the function/job
	// add the channel to the downstream vertex depCh map ->
	// used to wait for the upstream vertex to signal the fn/job is done
	for vertexName, wCtx := range r.walkMap {
		// only run these channels when we want to add dependency validation
		for _, depVertexName := range d.GetUpVertexes(vertexName) {
			//fmt.Printf("vertexName: %s, depBVertexName: %s\n", vertexName, depVertexName)
			depCh := make(chan bool)
			r.walkMap[depVertexName].AddDoneCh(vertexName, depCh) // send when done
			wCtx.AddDepCh(depVertexName, depCh)                   // rcvr when done
		}
		doneFnCh := make(chan bool)
		wCtx.doneFnCh = doneFnCh
		r.fnDoneMap[vertexName] = doneFnCh
	}
}

func (r *exectutor) walk(ctx context.Context, d dag.DAG, from string, init bool, depth int) {
	wCtx := r.getWalkContext(from)
	// avoid scheduling a vertex that is already visted
	if !wCtx.isVisted() {
		wCtx.m.Lock()
		wCtx.visited = time.Now()
		wCtx.m.Unlock()
		// execute the vertex function
		fmt.Printf("%s scheduled\n", wCtx.vertexName)
		go func() {
			if !r.dependenciesFinished(wCtx.depChs) {
				fmt.Printf("%s not finished\n", from)
			}
			if !wCtx.waitDependencies(ctx) {
				// TODO gather info why the failure occured
				return
			}
			// execute the vertex function
			wCtx.run(ctx)
		}()
	}
	// continue walking the graph
	depth++
	for _, downEdge := range d.GetDownVertexes(from) {
		go func(downEdge string) {
			r.walk(ctx, d, downEdge, false, depth)
		}(downEdge)
	}
	if init {
		r.waitFunctionCompletion(ctx)
	}
}

func (r *exectutor) getWalkContext(s string) *vertexContext {
	r.mw.RLock()
	defer r.mw.RUnlock()
	return r.walkMap[s]
}

func (r *exectutor) dependenciesFinished(dep map[string]chan bool) bool {
	for vertexName := range dep {
		if !r.getWalkContext(vertexName).isFinished() {
			return false
		}
	}
	return true
}

func (r *exectutor) waitFunctionCompletion(ctx context.Context) {
	fmt.Printf("main walk wait waiting for function completion...\n")
DepSatisfied:
	for vertexName, doneFnCh := range r.fnDoneMap {
		for {
			select {
			case d, ok := <-doneFnCh:
				fmt.Printf("main walk wait rcvd fn done from %s, d: %t, ok: %t\n", vertexName, d, ok)
				if !d {
					r.cancelFn()
					return
				}
				continue DepSatisfied
			case <-ctx.Done():
				// called when the controller gets cancelled
				return
			case <-time.After(time.Second * 5):
				fmt.Printf("main walk wait timeout, waiting for %s\n", vertexName)
			}
		}
	}
	fmt.Printf("main walk wait function completion waiting finished - bye !\n")
}
