package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/fnmap"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Executor interface {
	Run(ctx context.Context, req ctrl.Request)
	GetResult()
	GetOutput()
}

type executor struct {
	d     dag.DAG
	fnMap fnmap.FnMap
	req   ctrl.Request

	// cancelFn
	cancelFn context.CancelFunc

	// used during the Walk func
	mw        sync.RWMutex
	execMap   map[string]*execContext
	fnDoneMap map[string]chan bool

	mr         sync.RWMutex
	execResult []*result

	output Output
}

type Config struct {
	Client client.Client
	GVK    schema.GroupVersionKind
	DAG    dag.DAG
}

func New(cfg *Config) Executor {
	s := &executor{
		d: cfg.DAG,
		// fnMap contains the internal functions
		fnMap: fnmap.New(&fnmap.FnMapConfig{
			Client: cfg.Client,
			GVK:    cfg.GVK,
		}),
		mw:         sync.RWMutex{},
		execMap:    map[string]*execContext{},
		fnDoneMap:  map[string]chan bool{},
		mr:         sync.RWMutex{},
		execResult: []*result{},
		output:     NewOutput(),
	}
	s.init()
	return s
}

func (r *executor) init() {
	r.execResult = []*result{}
	r.execMap = map[string]*execContext{}
	for vertexName, dvc := range r.d.GetVertices() {
		//fmt.Printf("init vertexName: %s\n", vertexName)
		r.execMap[vertexName] = &execContext{
			vertexName:    vertexName,
			vertexContext: dvc,
			doneChs:       make(map[string]chan bool), //snd
			depChs:        make(map[string]chan bool), //rcv
			// callback to gather the result
			recordResult: r.recordResult,
			fnMap:        r.fnMap,
			output:       r.output,
		}
	}
	// build the channel matrix to signal dependencies through channels
	// for every dependency (upstream relationship between verteces)
	// create a channel
	// add the channel to the upstreamm vertex doneCh map ->
	// usedto signal/send the vertex finished the function/job
	// add the channel to the downstream vertex depCh map ->
	// used to wait for the upstream vertex to signal the fn/job is done
	for vertexName, wCtx := range r.execMap {
		// only run these channels when we want to add dependency validation
		for _, depVertexName := range r.d.GetUpVertexes(vertexName) {
			//fmt.Printf("vertexName: %s, depBVertexName: %s\n", vertexName, depVertexName)
			depCh := make(chan bool)
			r.execMap[depVertexName].AddDoneCh(vertexName, depCh) // send when done
			wCtx.AddDepCh(depVertexName, depCh)                   // rcvr when done
		}
		doneFnCh := make(chan bool)
		wCtx.doneFnCh = doneFnCh
		r.fnDoneMap[vertexName] = doneFnCh
	}
}

func (r *executor) Run(ctx context.Context, req ctrl.Request) {
	r.req = req
	from := r.d.GetRootVertex()
	start := time.Now()
	ctx, cancelFn := context.WithCancel(ctx)
	r.cancelFn = cancelFn
	r.execute(ctx, req, from, true)
	// add total as a last entry in the result
	r.recordResult(&result{
		vertexName: "total",
		startTime:  start,
		endTime:    time.Now(),
	})
}

func (r *executor) execute(ctx context.Context, req ctrl.Request, from string, init bool) {
	wCtx := r.getExecContext(from)
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
			wCtx.run(ctx, req)
		}()
	}
	// continue walking the graph
	for _, downEdge := range r.d.GetDownVertexes(from) {
		go func(downEdge string) {
			r.execute(ctx, req, downEdge, false)
		}(downEdge)
	}
	if init {
		r.waitFunctionCompletion(ctx)
	}
}

func (r *executor) getExecContext(s string) *execContext {
	r.mw.RLock()
	defer r.mw.RUnlock()
	return r.execMap[s]
}

func (r *executor) dependenciesFinished(dep map[string]chan bool) bool {
	for vertexName := range dep {
		if !r.getExecContext(vertexName).isFinished() {
			return false
		}
	}
	return true
}

func (r *executor) waitFunctionCompletion(ctx context.Context) {
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
