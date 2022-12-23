package ccsyntax

import (
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// cfgPreHookFn processes the for, own, watch generically
type cfgPreHookFn func(lcncCfg *ctrlcfgv1.ControllerConfig)
type cfgPostHookFn func(lcncCfg *ctrlcfgv1.ControllerConfig)

// gvkObjectFn processes the for, own, watch per item
type gvkObjectFn func(oc *OriginContext, v *ctrlcfgv1.GvkObject) *schema.GroupVersionKind
type emptyPipelineFn func(oc *OriginContext, v *ctrlcfgv1.GvkObject)

// lcncBlockFn processes the block part of the Variables and functions
//type pipelineBlockFn func(o Origin, idx int, vertexName string, v ctrlcfgv1.ControllerConfigBlock)
//type pipelineBlockEndFn func(o Origin, idx int, vertexName string, v ctrlcfgv1.ControllerConfigBlock)

//type lcncVarsPreHookFn func(v []ctrlcfgv1.ControllerConfigVarBlock)
//type lcncVarsPostHookFn func(v []ctrlcfgv1.ControllerConfigVarBlock)

// lcncVarFn processes the variable in the variables section
//type lcncVarFn func(o Origin, block bool, idx int, vertexName string, v ctrlcfgv1.ControllerConfigVar)

type pipelinePreHookFn func(oc *OriginContext, v *ctrlcfgv1.Pipeline)
type pipelinePostHookFn func(oc *OriginContext, v *ctrlcfgv1.Pipeline)

// functionFn processes the function in the functions section
type functionFn func(oc *OriginContext, v *ctrlcfgv1.Function)

//type lcncServicesPreHookFn func(v []ctrlcfgv1.ControllerConfigFunctionsBlock)

//type lcncServicesPostHookFn func(v []LcncFunctionsBlock)

// lcncServiceFn processes the service in the services section
//type lcncServiceFn func(o Origin, block bool, idx int, vertexName string, v ctrlcfgv1.ControllerConfigFunction)

type WalkConfig struct {
	cfgPreHookFn    cfgPreHookFn
	cfgPostHookFn   cfgPostHookFn
	gvkObjectFn     gvkObjectFn
	emptyPipelineFn emptyPipelineFn

	pipelinePreHookFn  pipelinePreHookFn
	functionFn         functionFn
	pipelinePostHookFn pipelinePostHookFn
	//pipelineBlockFn    pipelineBlockFn
	//pipelineBlockEndFn pipelineBlockEndFn
	//lcncServicesPreHookFn   lcncServicesPreHookFn
	//lcncServiceFn           lcncServiceFn
	//lcncServicesPostHookFn  lcncServicesPreHookFn
}

func (r *parser) walkLcncConfig(fnc *WalkConfig) {
	// process config entry
	if fnc.cfgPreHookFn != nil {
		fnc.cfgPreHookFn(r.cCfg)
	}

	// process for, own, watch
	idx := 0
	for vertexName, v := range r.cCfg.Spec.Properties.For {
		// we run this once for apply and once for delete
		oc := &OriginContext{FOW: FOWFor, Origin: OriginFow, VertexName: vertexName}
		r.processGvkObject(fnc, oc, v)
		idx++

	}
	idx = 0
	for vertexName, v := range r.cCfg.Spec.Properties.Own {
		// For Own the oepration is irrelevant
		oc := &OriginContext{FOW: FOWOwn, Origin: OriginFow, VertexName: vertexName}
		r.processGvkObject(fnc, oc, v)
		idx++
	}
	idx = 0
	for vertexName, v := range r.cCfg.Spec.Properties.Watch {
		// we run this only for operation apply, NOT for delete
		oc := &OriginContext{FOW: FOWWatch, Origin: OriginFow, VertexName: vertexName}
		r.processGvkObject(fnc, oc, v)
	}

	if fnc.cfgPostHookFn != nil {
		fnc.cfgPostHookFn(r.cCfg)
	}
}

func (r *parser) processGvkObject(fnc *WalkConfig, oc *OriginContext, v *ctrlcfgv1.GvkObject) {
	if fnc.gvkObjectFn != nil {
		gvk := fnc.gvkObjectFn(oc, v)
		oc.GVK = gvk
		oc.Operation = OperationApply
		applyPipeline := r.cCfg.GetPipeline(v.ApplyPipelineRef)
		if applyPipeline == nil {
			if fnc.emptyPipelineFn != nil {
				fnc.emptyPipelineFn(oc, v)
			}
		} else {
			fnc.walkPipeline(oc, applyPipeline)
		}

		oc.Operation = OperationDelete
		deletePipeline := r.cCfg.GetPipeline(v.DeletePipelineRef)
		if deletePipeline == nil {
			if fnc.emptyPipelineFn != nil {
				fnc.emptyPipelineFn(oc, v)
			}
		} else {
			fnc.walkPipeline(oc, deletePipeline)
		}
	}
}

func (fnc *WalkConfig) walkPipeline(oc *OriginContext, v *ctrlcfgv1.Pipeline) {
	pipelineName := v.Name
	if fnc.pipelinePreHookFn != nil {
		oc := &OriginContext{
			FOW:        oc.FOW,
			Operation:  oc.Operation,
			GVK:        oc.GVK,
			Pipeline:   pipelineName,
			Origin:     oc.Origin,
			VertexName: oc.VertexName,
		}
		fnc.pipelinePreHookFn(oc, v)
	}

	for vertexName, v := range v.Vars {
		if fnc.functionFn != nil {
			oc := &OriginContext{
				FOW:        oc.FOW,
				Operation:  oc.Operation,
				GVK:        oc.GVK,
				Pipeline:   pipelineName,
				Origin:     OriginVariable,
				VertexName: vertexName,
			}
			fnc.functionFn(oc, v)
		}
	}

	for vertexName, v := range v.Tasks {
		if fnc.functionFn != nil {
			oc := &OriginContext{
				FOW:        oc.FOW,
				Operation:  oc.Operation,
				GVK:        oc.GVK,
				Pipeline:   pipelineName,
				Origin:     OriginFunction,
				VertexName: vertexName,
			}
			fnc.functionFn(oc, v)
		}
	}

	if fnc.pipelinePostHookFn != nil {
		oc := &OriginContext{
			FOW:        oc.FOW,
			Operation:  oc.Operation,
			GVK:        oc.GVK,
			Pipeline:   pipelineName,
			Origin:     oc.Origin,
			VertexName: oc.VertexName,
		}
		fnc.pipelinePostHookFn(oc, v)
	}
}
