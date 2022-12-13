package ccsyntax

import (
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Parser interface {
	GetExternalResources() ([]schema.GroupVersionResource, []Result)
	Parse() (dag.DAG, string, []Result)
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
	cCfg *ctrlcfgv1.ControllerConfig
	//d              dag.DAG
	rootVertexName string
}

func (r *parser) Parse() (dag.DAG, string, []Result) {
	// validate the config when creating the dag
	d := dag.New()
	// resolves the dependencies in the dag
	// step1. check if all dependencies resolve
	// step2. add the dependencies in the dag
	result := r.populate(d)
	if len(result) != 0 {
		return nil, "", result
	}
	result = r.resolve(d)
	if len(result) != 0 {
		return nil, "", result
	}
	result = r.connect(d)
	if len(result) != 0 {
		return nil, "", result
	}
	//d.GetDependencyMap(r.rootVertexName)
	// optimizes the dependncy graph based on transit reduction
	// techniques
	d.TransitiveReduction()
	//d.GetDependencyMap(r.rootVertexName)
	return d, r.rootVertexName, nil
}
