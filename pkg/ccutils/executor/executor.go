package executor

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/yndd/lcnc-runtime/pkg/ccutils/dag"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Executor interface {
	Run(ctx context.Context)
}

type VertexFuntionRunFn func(ctx context.Context, vertexName string, vertexContext any) bool
type ExecPostRunFn func(start, finish time.Time, success bool)

type Config struct {
	Name string
	From string
	//Handlers
	VertexFuntionRunFn VertexFuntionRunFn
	ExecPostRunFn      ExecPostRunFn
}

func New(d dag.DAG, cfg *Config) Executor {
	s := &exec{
		cfg:       cfg,
		d:         d,
		m:         sync.RWMutex{},
		execMap:   map[string]*execContext{},
		fnDoneMap: map[string]chan bool{},

		l: ctrl.Log.WithName("executor"),
	}

	// initialize the initial data in the executor
	s.init()
	return s
}

type exec struct {
	d   dag.DAG
	cfg *Config

	// cancelFn
	cancelFn context.CancelFunc

	// used during the Walk func
	m         sync.RWMutex
	execMap   map[string]*execContext
	fnDoneMap map[string]chan bool
	// logging
	l logr.Logger
}

// init initializes the executor with channels and cancel context
// so it is prepaared to execute the dependency map
func (r *exec) init() {
	r.execMap = map[string]*execContext{}
	for vertexName, v := range r.d.GetVertices() {
		//fmt.Printf("executor init vertexName: %s\n", vertexName)
		r.l.Info("init", "vertexName", vertexName)
		r.execMap[vertexName] = &execContext{
			execName:      r.cfg.Name,
			vertexName:    vertexName,
			vertexContext: v,
			doneChs:       make(map[string]chan bool), //snd
			depChs:        make(map[string]chan bool), //rcv
			deps:          make([]string, 0),
			// callback to gather the result
			//recordResult: r.cfg.Result.Add,
			//recordOutput: r.cfg.Output.Add,
			vertexFuntionRunFn: r.cfg.VertexFuntionRunFn,
			l:                  ctrl.Log.WithName("execContext").WithValues("execName", r.cfg.Name, "vertexName", vertexName),
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
		wCtx.deps = r.d.GetUpVertexes(vertexName)
		doneFnCh := make(chan bool)
		wCtx.doneFnCh = doneFnCh
		r.fnDoneMap[vertexName] = doneFnCh
	}
}

func (r *exec) Run(ctx context.Context) {
	from := r.cfg.From
	start := time.Now()
	ctx, cancelFn := context.WithCancel(ctx)
	r.cancelFn = cancelFn
	success := r.execute(ctx, from, true)
	finish := time.Now()

	// handler to execute a final action e.g. recording the overall result
	r.cfg.ExecPostRunFn(start, finish, success)
}

func (r *exec) execute(ctx context.Context, from string, init bool) bool {
	r.l.Info("execute", "from", from, "init", init)
	//fmt.Printf("execute from: %s init: %t\n", from, init)
	wCtx := r.getExecContext(from)
	//fmt.Printf("execute getExecContext from: %s init: %t, wCtx: %#v\n", from, init, wCtx)
	// avoid scheduling a vertex that is already visted
	if !wCtx.isVisted() {
		wCtx.m.Lock()
		wCtx.visited = time.Now()
		wCtx.m.Unlock()
		// execute the vertex function
		r.l.Info("execute scheduled vertex", "vertexname", wCtx.vertexName)
		//fmt.Printf("execute scheduled vertex: %s\n", wCtx.vertexName)
		go func() {
			if !r.dependenciesFinished(wCtx.depChs) {
				//fmt.Printf("%s not finished\n", from)
				r.l.Info("not finished", "vertexname", from)
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
	//fmt.Printf("main walk wait waiting for function completion...\n")
	r.l.Info("main walk wait waiting for function completion...")
DepSatisfied:
	for vertexName, doneFnCh := range r.fnDoneMap {
		for {
			select {
			case d, ok := <-doneFnCh:
				r.l.Info("main walk wait rcvd fn done", "from", vertexName, "success", d, "ok", ok)
				//fmt.Printf("main walk wait rcvd fn done from %s, d: %t, ok: %t\n", vertexName, d, ok)
				if !d {
					r.cancelFn()
					return false
				}
				continue DepSatisfied
			case <-ctx.Done():
				// called when the controller gets cancelled
				return false
			case <-time.After(time.Second * 5):
				r.l.Info("main walk wait timeout, waiting", "for", vertexName)
				//fmt.Printf("main walk wait timeout, waiting for %s\n", vertexName)
			}
		}
	}
	r.l.Info("main walk wait function completion waiting finished - bye !")
	//fmt.Printf("main walk wait function completion waiting finished - bye !\n")
	return true
}
