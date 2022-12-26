package builder

import (
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/executor"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap/functions"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
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
	Output         output.Output
	Result         result.Result
}

func New(c *Config) executor.Executor {
	// create a new output

	// create a new fn map
	fnmap := functions.Init(&fnmap.Config{
		Name:      c.Name,
		Namespace: c.Namespace,
		Client:    c.Client,
		Output:    c.Output,
		Result:    c.Result,
	})

	// Initialize the initial data
	c.Output.RecordOutput(c.RootVertexName, &output.OutputInfo{
		Internal: true,
		Value:    c.Data,
	})

	return executor.New(&executor.Config{
		Type:           result.ExecRootType,
		Name:           c.DAG.GetRootVertex(),
		RootVertexName: c.RootVertexName,
		DAG:            c.DAG,
		FnMap:          fnmap,
		Output:         c.Output,
		Result:         c.Result,
	})
}
