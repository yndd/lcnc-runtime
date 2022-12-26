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
	Run(ctx context.Context)
}

type exec struct {
	cfg *Config

	// cancelFn
	cancelFn context.CancelFunc

	// used during the Walk func
	m         sync.RWMutex
	execMap   map[string]*execContext
	fnDoneMap map[string]chan bool
}

type Config struct {
	Type           result.ExecType
	Name           string
	RootVertexName string
	DAG            dag.DAG
	FnMap          fnmap.FuncMap
	Output         output.Output
	Result         result.Result
}

func New(cfg *Config) Executor {
	s := &exec{
		cfg:       cfg,
		m:         sync.RWMutex{},
		execMap:   map[string]*execContext{},
		fnDoneMap: map[string]chan bool{},
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
	for vertexName, dvc := range r.cfg.DAG.GetVertices() {
		fmt.Printf("executor init vertexName: %s\n", vertexName)
		r.execMap[vertexName] = &execContext{
			cfg:           r.cfg,
			vertexName:    vertexName,
			vertexContext: dvc,
			doneChs:       make(map[string]chan bool), //snd
			depChs:        make(map[string]chan bool), //rcv
			// callback to gather the result
			recordResult: r.cfg.Result.RecordResult,
			recordOutput: r.cfg.Output.RecordOutput,
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
		for _, depVertexName := range r.cfg.DAG.GetUpVertexes(vertexName) {
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

func (r *exec) Run(ctx context.Context) {
	from := r.cfg.DAG.GetRootVertex()
	start := time.Now()
	ctx, cancelFn := context.WithCancel(ctx)
	r.cancelFn = cancelFn
	success := r.execute(ctx, from, true)
	// add total as a last entry in the result
	r.cfg.Result.RecordResult(&result.ResultInfo{
		Type:       r.cfg.Type,
		ExecName:   r.cfg.Name,
		VertexName: "total",
		StartTime:  start,
		EndTime:    time.Now(),
		Success:    success,
	})
}

func (r *exec) execute(ctx context.Context, from string, init bool) bool {
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
	for _, downEdge := range r.cfg.DAG.GetDownVertexes(from) {
		go func(downEdge string) {
			r.execute(ctx, downEdge, false)
		}(downEdge)
	}
	if init {
		return r.waitFunctionCompletion(ctx)
	}
	return true
}

func (r *exec) getExecContext(s string) *execContext {
	r.m.RLock()
	defer r.m.RUnlock()
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

func (r *exec) waitFunctionCompletion(ctx context.Context) bool {
	fmt.Printf("main walk wait waiting for function completion...\n")
DepSatisfied:
	for vertexName, doneFnCh := range r.fnDoneMap {
		for {
			select {
			case d, ok := <-doneFnCh:
				fmt.Printf("main walk wait rcvd fn done from %s, d: %t, ok: %t\n", vertexName, d, ok)
				if !d {
					r.cancelFn()
					return false
				}
				continue DepSatisfied
			case <-ctx.Done():
				// called when the controller gets cancelled
				return false
			case <-time.After(time.Second * 5):
				fmt.Printf("main walk wait timeout, waiting for %s\n", vertexName)
			}
		}
	}
	fmt.Printf("main walk wait function completion waiting finished - bye !\n")
	return true
}
