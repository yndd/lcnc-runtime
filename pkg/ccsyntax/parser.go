package ccsyntax

import (
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Parser interface {
	GetExternalResources() ([]*schema.GroupVersionKind, []Result)
	Parse() (ConfigExecutionContext, []Result)
}

func NewParser(cfg *ctrlcfgv1.ControllerConfig) (Parser, []Result) {
	p := &parser{
		cCfg: cfg,
		//d:       dag.NewDAG(),
		//output: map[string]string{},
	}
	// add the callback function to record validation results results
	result := p.ValidateSyntax()
	p.rootVertexName = cfg.GetRootVertexName()

	return p, result
}

type parser struct {
	cCfg           *ctrlcfgv1.ControllerConfig
	rootVertexName string
}

func (r *parser) Parse() (ConfigExecutionContext, []Result) {
	// initialize the config execution context
	// for each for and watch a new dag is created
	ceCtx, gvar, result := r.init()
	if len(result) != 0 {
		return nil, result
	}
	// resolves the dependencies in the dag
	// step1. check if all dependencies resolve
	// step2. add the dependencies in the dag
	result = r.populate(ceCtx, gvar)
	if len(result) != 0 {
		return nil, result
	}
	//fmt.Println("propulate succeded")
	result = r.resolve(ceCtx, gvar)
	if len(result) != 0 {
		return nil, result
	}
	//fmt.Println("resolve succeded")
	result = r.connect(ceCtx, gvar)
	if len(result) != 0 {
		return nil, result
	}
	// optimizes the dependncy graph based on transit reduction
	// techniques
	r.transitivereduction(ceCtx)

	ceCtx.Print()
	return ceCtx, nil
}

func (r *parser) transitivereduction(ceCtx ConfigExecutionContext) {
	// transitive reduction for For dag
	for _, od := range ceCtx.GetFOW(FOWFor) {
		for _, dctx := range od {
			dctx.DAG.TransitiveReduction()
			for _, d := range dctx.BlockDAGs {
				d.TransitiveReduction()
			}
		}

	}
	// transitive reduction for Watch dags
	for _, od := range ceCtx.GetFOW(FOWWatch) {
		for _, dctx := range od {
			dctx.DAG.TransitiveReduction()
			for _, d := range dctx.BlockDAGs {
				d.TransitiveReduction()
			}
		}
	}
}
