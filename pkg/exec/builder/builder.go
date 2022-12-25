package builder

import (
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/executor"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	Name           string
	Namespace      string
	RootVertexName string
	Data           any
	Client         client.Client
	GVK            *schema.GroupVersionKind
	DAG            dag.DAG
}

func New(cfg *Config) (executor.Executor, output.Output) {
	// create a new output
	o := output.New()

	// create a new fn map
	fnm := fnmap.New(&fnmap.Config{
		Name:      cfg.Name,
		Namespace: cfg.Namespace,
		Client:    cfg.Client,
		GVK:       cfg.GVK,
		Output:    o,
	})

	// Initialize the initial data
	o.RecordOutput(cfg.RootVertexName, &output.OutputInfo{
		Internal: true,
		Value:    cfg.Data,
	})

	return executor.New(&executor.Config{
		RootVertexName: cfg.RootVertexName,
		DAG:            cfg.DAG,
		FnMap:          fnm,
		Output:         o,
	}), o
}
