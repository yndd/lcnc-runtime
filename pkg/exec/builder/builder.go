package builder

import (
	"github.com/yndd/lcnc-runtime/pkg/exec/exechandler"
	"github.com/yndd/lcnc-runtime/pkg/ccutils/executor"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap/functions"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/exec/rtdag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	Name      string
	Namespace string
	Data      any
	Client    client.Client
	GVK       *schema.GroupVersionKind
	DAG       rtdag.RuntimeDAG
	Output    output.Output
	Result    result.Result
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

	rootVertexName := c.DAG.GetRootVertex()

	// Initialize the initial data
	c.Output.AddEntry(rootVertexName, &output.OutputInfo{
		Internal: true,
		GVK:      c.GVK,
		Data:     c.Data,
	})

	// initialize the handler
	h := exechandler.New(&exechandler.Config{
		Name:   rootVertexName,
		Type:   result.ExecRootType,
		DAG:    c.DAG,
		FnMap:  fnmap,
		Output: c.Output,
		Result: c.Result,
	})

	return executor.New(c.DAG, &executor.Config{
		Name:               rootVertexName,
		From:               rootVertexName,
		VertexFuntionRunFn: h.FunctionRun,
		ExecPostRunFn:      h.RecordFinalResult,
	})
}
