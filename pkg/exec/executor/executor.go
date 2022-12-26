package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
)

type Executor interface {
	Run(ctx context.Context) result.Result
	//PrintResult()
}

type exec struct {
	name           string
	rootVertexName string
	d              dag.DAG
	fnMap          fnmap.FuncMap
	output         output.Output
	result         result.Result

	// cancelFn
	cancelFn context.CancelFunc

	// used during the Walk func
	mw        sync.RWMutex
	execMap   map[string]*execContext
	fnDoneMap map[string]chan bool

	//mr         sync.RWMutex
	//execResult []*result
}

type Config struct {
	Name           string
	RootVertexName string
	//Data       any
	//Client     client.Client
	//GVK        *schema.GroupVersionKind
	DAG    dag.DAG
	FnMap  fnmap.FuncMap
	Output output.Output
}

func New(cfg *Config) Executor {
	s := &exec{
		name:           cfg.Name,
		rootVertexName: cfg.RootVertexName,
		d:              cfg.DAG,
		fnMap:          cfg.FnMap,
		output:         cfg.Output,
		result:         result.New(),
		mw:             sync.RWMutex{},
		execMap:        map[string]*execContext{},
		fnDoneMap:      map[string]chan bool{},
	}

	// initialize the initial data in the executor
	//s.output.Update(cfg.RootVertex, cfg.RootVertex, &fnmap.Output{Internal: true, Value: cfg.Data})
	s.init()
	return s
}

// init initializes the executor with channels and cancel context
// so it is prepaared to execute the dependency map
func (r *exec) init() {
	r.execMap = map[string]*execContext{}
	for vertexName, dvc := range r.d.GetVertices() {
		fmt.Printf("executor init vertexName: %s\n", vertexName)
		r.execMap[vertexName] = &execContext{
			execName:       r.name,
			vertexName:     vertexName,
			rootVertexName: r.rootVertexName,
			vertexContext:  dvc,
			doneChs:        make(map[string]chan bool), //snd
			depChs:         make(map[string]chan bool), //rcv
			// callback to gather the result
			recordResult: r.result.RecordResult,
			recordOutput: r.output.RecordOutput,
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

func (r *exec) Run(ctx context.Context) result.Result {
	from := r.d.GetRootVertex()
	start := time.Now()
	ctx, cancelFn := context.WithCancel(ctx)
	r.cancelFn = cancelFn
	r.execute(ctx, from, true)
	// add total as a last entry in the result
	r.result.RecordResult(&result.ResultInfo{
		VertexName: "total",
		StartTime:  start,
		EndTime:    time.Now(),
	})

	return r.result
}

func (r *exec) execute(ctx context.Context, from string, init bool) {
	fmt.Printf("execute from: %s init: %t\n", from, init)
	wCtx := r.getExecContext(from)
	fmt.Printf("execute getExecContext from: %s init: %t, wCtx: %#v\n", from, init, wCtx)
	// avoid scheduling a vertex that is already visted
	if !wCtx.isVisted() {
		wCtx.m.Lock()
		wCtx.visited = time.Now()
		wCtx.m.Unlock()
		// execute the vertex function
		fmt.Printf("execute scheduled vertex: %s\n", wCtx.vertexName)
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
	for _, downEdge := range r.d.GetDownVertexes(from) {
		go func(downEdge string) {
			r.execute(ctx, downEdge, false)
		}(downEdge)
	}
	if init {
		r.waitFunctionCompletion(ctx)
	}
}

func (r *exec) getExecContext(s string) *execContext {
	r.mw.RLock()
	defer r.mw.RUnlock()
	return r.execMap[s]
}

func (r *exec) dependenciesFinished(dep map[string]chan bool) bool {
	for vertexName := range dep {
		if !r.getExecContext(vertexName).isFinished() {
			return false
		}
	}
	return true
}

func (r *exec) waitFunctionCompletion(ctx context.Context) {
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
