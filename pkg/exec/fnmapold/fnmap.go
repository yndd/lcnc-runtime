package fnmap

import (
	"context"

	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FnMap interface {
	RunFn(ctx context.Context, vertexContext *dag.VertexContext, input map[string]any) (map[string]*output.OutputInfo, error)
}

type Config struct {
	Name           string
	Namespace      string
	RootVertexName string
	Client         client.Client
	GVK            *schema.GroupVersionKind
	Output         output.Output
}

func New(cfg *Config) FnMap {
	f := &fnmap{
		name:           cfg.Name,
		namespace:      cfg.Namespace,
		rootVertexName: cfg.RootVertexName,
		client:         cfg.Client,
		gvk:            cfg.GVK,
		output:         cfg.Output,
	}
	return f
}

type fnmap struct {
	name           string
	namespace      string
	rootVertexName string
	client         client.Client
	gvk            *schema.GroupVersionKind
	output         output.Output
}

//type Result map[string]*output.OutputInfo
