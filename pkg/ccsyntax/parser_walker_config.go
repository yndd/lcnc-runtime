package ccsyntax

import (
	"fmt"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

// cfgPreHookFn processes the for, own, watch generically
type cfgPreHookFn func(lcncCfg *ctrlcfgv1.ControllerConfig)
type cfgPostHookFn func(lcncCfg *ctrlcfgv1.ControllerConfig)

// gvrObjectFn processes the for, own, watch per item
type gvrObjectFn func(o Origin, idx int, vertexName string, v ctrlcfgv1.ControllerConfigGvrObject)

// lcncBlockFn processes the block part of the Variables and functions
type pipelineBlockFn func(o Origin, idx int, vertexName string, v ctrlcfgv1.ControllerConfigBlock)

//type lcncVarsPreHookFn func(v []ctrlcfgv1.ControllerConfigVarBlock)
//type lcncVarsPostHookFn func(v []ctrlcfgv1.ControllerConfigVarBlock)

// lcncVarFn processes the variable in the variables section
//type lcncVarFn func(o Origin, block bool, idx int, vertexName string, v ctrlcfgv1.ControllerConfigVar)

type pipelinePreHookFn func(v ctrlcfgv1.ControllerConfigPipeline)
type pipelinePostHookFn func(v ctrlcfgv1.ControllerConfigPipeline)

// functionFn processes the function in the functions section
type functionFn func(o Origin, idx int, vertexName string, v ctrlcfgv1.ControllerConfigFunction)

//type lcncServicesPreHookFn func(v []ctrlcfgv1.ControllerConfigFunctionsBlock)

//type lcncServicesPostHookFn func(v []LcncFunctionsBlock)

// lcncServiceFn processes the service in the services section
//type lcncServiceFn func(o Origin, block bool, idx int, vertexName string, v ctrlcfgv1.ControllerConfigFunction)

type WalkConfig struct {
	cfgPreHookFn  cfgPreHookFn
	cfgPostHookFn cfgPostHookFn
	gvrObjectFn   gvrObjectFn

	pipelinePreHookFn  pipelinePreHookFn
	pipelineBlockFn    pipelineBlockFn
	functionFn         functionFn
	pipelinePostHookFn pipelinePostHookFn
	//lcncServicesPreHookFn   lcncServicesPreHookFn
	//lcncServiceFn           lcncServiceFn
	//lcncServicesPostHookFn  lcncServicesPreHookFn
}

func (r *lcncparser) walkLcncConfig(fnc *WalkConfig) {
	// process config entry
	if fnc.cfgPreHookFn != nil {
		fnc.cfgPreHookFn(r.lcncCfg)
	}

	// process for, own, watch
	if fnc.gvrObjectFn != nil {
		idx := 0
		for vertexName, v := range r.lcncCfg.Spec.Properties.For {
			fnc.gvrObjectFn(OriginFor, idx, vertexName, v)
			fnc.walkPipeline(OriginFor, idx, vertexName, v.ControllerConfigPipeline)
			idx++

		}
		idx = 0
		for vertexName, v := range r.lcncCfg.Spec.Properties.Own {
			fnc.gvrObjectFn(OriginOwn, idx, vertexName, v)
			fnc.walkPipeline(OriginOwn, idx, vertexName, v.ControllerConfigPipeline)
			idx++
		}
		idx = 0
		for vertexName, v := range r.lcncCfg.Spec.Properties.Watch {
			fnc.gvrObjectFn(OriginWatch, idx, vertexName, v)
			fnc.walkPipeline(OriginWatch, idx, vertexName, v.ControllerConfigPipeline)
		}
	}
}

func (fnc *WalkConfig) walkPipeline(o Origin, idx int, parentVertexName string, v ctrlcfgv1.ControllerConfigPipeline) {
	if fnc.pipelinePreHookFn != nil {
		fnc.pipelinePreHookFn(v)
	}

	for vertexName, v := range v.Functions {
		if v.IsBlock() {
			fnc.pipelineBlockFn(o, idx, vertexName, v.ControllerConfigBlock)
			// execute the remaining part as a block
			fnc.walkPipeline(o, idx, vertexName, v.ControllerConfigPipeline)
			continue
		}
		// excute the function
		if fnc.functionFn != nil {
			fnc.functionFn(o, idx, vertexName, v.ControllerConfigFunction)
		}
	}
	if fnc.pipelinePostHookFn != nil {
		fnc.pipelinePostHookFn(v)
	}
}
