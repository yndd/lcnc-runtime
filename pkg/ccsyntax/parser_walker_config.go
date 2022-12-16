package ccsyntax

import (
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

// cfgPreHookFn processes the for, own, watch generically
type cfgPreHookFn func(lcncCfg *ctrlcfgv1.ControllerConfig)
type cfgPostHookFn func(lcncCfg *ctrlcfgv1.ControllerConfig)

// gvkObjectFn processes the for, own, watch per item
type gvkObjectFn func(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvkObject)
type emptyPipelineFn func(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvkObject)

// lcncBlockFn processes the block part of the Variables and functions
//type pipelineBlockFn func(o Origin, idx int, vertexName string, v ctrlcfgv1.ControllerConfigBlock)
//type pipelineBlockEndFn func(o Origin, idx int, vertexName string, v ctrlcfgv1.ControllerConfigBlock)

//type lcncVarsPreHookFn func(v []ctrlcfgv1.ControllerConfigVarBlock)
//type lcncVarsPostHookFn func(v []ctrlcfgv1.ControllerConfigVarBlock)

// lcncVarFn processes the variable in the variables section
//type lcncVarFn func(o Origin, block bool, idx int, vertexName string, v ctrlcfgv1.ControllerConfigVar)

type pipelinePreHookFn func(oc *OriginContext, v *ctrlcfgv1.ControllerConfigPipeline)
type pipelinePostHookFn func(oc *OriginContext, v *ctrlcfgv1.ControllerConfigPipeline)

// functionFn processes the function in the functions section
type functionFn func(oc *OriginContext, v *ctrlcfgv1.ControllerConfigFunction)

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
		oc := &OriginContext{FOW: FOWFor, Origin: OriginFow, VertexName: vertexName}
		r.processGvkObject(fnc, oc, v)
		idx++

	}
	idx = 0
	for vertexName, v := range r.cCfg.Spec.Properties.Own {
		oc := &OriginContext{FOW: FOWOwn, Origin: OriginFow, VertexName: vertexName}
		r.processGvkObject(fnc, oc, v)
		idx++
	}
	idx = 0
	for vertexName, v := range r.cCfg.Spec.Properties.Watch {
		oc := &OriginContext{FOW: FOWWatch, Origin: OriginFow, VertexName: vertexName}
		r.processGvkObject(fnc, oc, v)
	}

	if fnc.cfgPostHookFn != nil {
		fnc.cfgPostHookFn(r.cCfg)
	}

}

func (r *parser) processGvkObject(fnc *WalkConfig, oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvkObject) {
	if fnc.gvkObjectFn != nil {
		fnc.gvkObjectFn(oc, v)
		pipeline := r.cCfg.GetPipeline(v.PipelineRef)
		if pipeline == nil {
			if fnc.emptyPipelineFn != nil {
				fnc.emptyPipelineFn(oc, v)
				return
			}
		}
		fnc.walkPipeline(oc, pipeline)
	}
}

func (fnc *WalkConfig) walkPipeline(oc *OriginContext, v *ctrlcfgv1.ControllerConfigPipeline) {
	pipelineName := v.Name
	if fnc.pipelinePreHookFn != nil {
		oc := &OriginContext{
			FOW:        oc.FOW,
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
			Pipeline:   pipelineName,
			Origin:     oc.Origin,
			VertexName: oc.VertexName,
		}
		fnc.pipelinePostHookFn(oc, v)
	}
}
