package ccsyntax

import (
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

// lcncCfgPreHookFn processes the for, own, watch generically
type lcncCfgPreHookFn func(lcncCfg *ctrlcfgv1.ControllerConfig)
type lcncCfgPostHookFn func(lcncCfg *ctrlcfgv1.ControllerConfig)

// lcncGvrObjectFn processes the for, own, watch per item
type lcncGvrObjectFn func(o Origin, idx int, n string, v ctrlcfgv1.ControllerConfigGvrObject)

// lcncBlockFn processes the block part of the Variables and functions
type lcncBlockFn func(o Origin, idx int, v ctrlcfgv1.ControllerConfigBlock)

type lcncVarsPreHookFn func(v []ctrlcfgv1.ControllerConfigVarBlock)
type lcncVarsPostHookFn func(v []ctrlcfgv1.ControllerConfigVarBlock)

// lcncVarFn processes the variable in the variables section
type lcncVarFn func(o Origin, block bool, idx int, vertexName string, v ctrlcfgv1.ControllerConfigVar)

type lcncFunctionsPreHookFn func(v []ctrlcfgv1.ControllerConfigFunctionsBlock)
type lcncFunctionsPostHookFn func(v []ctrlcfgv1.ControllerConfigFunctionsBlock)

// lcncFunctionFn processes the function in the functions section
type lcncFunctionFn func(o Origin, block bool, idx int, vertexName string, v ctrlcfgv1.ControllerConfigFunction)

type lcncServicesPreHookFn func(v []ctrlcfgv1.ControllerConfigFunctionsBlock)

//type lcncServicesPostHookFn func(v []LcncFunctionsBlock)

// lcncServiceFn processes the service in the services section
type lcncServiceFn func(o Origin, block bool, idx int, vertexName string, v ctrlcfgv1.ControllerConfigFunction)

type WalkConfig struct {
	lcncCfgPreHookFn        lcncCfgPreHookFn
	lcncCfgPostHookFn       lcncCfgPostHookFn
	lcncGvrObjectFn         lcncGvrObjectFn
	lcncBlockFn             lcncBlockFn
	lcncVarsPreHookFn       lcncVarsPreHookFn
	lcncVarFn               lcncVarFn
	lcncVarsPostHookFn      lcncVarsPostHookFn
	lcncFunctionsPreHookFn  lcncFunctionsPreHookFn
	lcncFunctionFn          lcncFunctionFn
	lcncFunctionsPostHookFn lcncFunctionsPostHookFn
	lcncServicesPreHookFn   lcncServicesPreHookFn
	lcncServiceFn           lcncServiceFn
	lcncServicesPostHookFn  lcncServicesPreHookFn
}

func (r *lcncparser) walkLcncConfig(fnc WalkConfig) {
	// process config entry
	if fnc.lcncCfgPreHookFn != nil {
		fnc.lcncCfgPreHookFn(r.lcncCfg)
	}

	// process for, own, watch
	if fnc.lcncGvrObjectFn != nil {
		idx := 0
		for vertexName, v := range r.lcncCfg.Spec.Properties.For {
			fnc.lcncGvrObjectFn(OriginFor, idx, vertexName, v)
			idx++
		}
		idx = 0
		for vertexName, v := range r.lcncCfg.Spec.Properties.Own {
			fnc.lcncGvrObjectFn(OriginOwn, idx, vertexName, v)
			idx++
		}
		idx = 0
		for vertexName, v := range r.lcncCfg.Spec.Properties.Watch {
			fnc.lcncGvrObjectFn(OriginWatch, idx, vertexName, v)
		}
	}

	// process variables
	if fnc.lcncVarsPreHookFn != nil {
		fnc.lcncVarsPreHookFn(r.lcncCfg.Spec.Properties.Vars)
	}
	for idx, vars := range r.lcncCfg.Spec.Properties.Vars {
		// check if there is a block
		block := false
		if vars.ControllerConfigBlock.For != nil {
			block = true
			if fnc.lcncBlockFn != nil {
				fnc.lcncBlockFn(OriginVariable, idx, vars.ControllerConfigBlock)
			}
		}
		for vertexName, v := range vars.ControllerConfigVariables {
			if fnc.lcncVarFn != nil {
				fnc.lcncVarFn(OriginVariable, block, idx, vertexName, v)
			}
		}
	}
	if fnc.lcncVarsPostHookFn != nil {
		fnc.lcncVarsPostHookFn(r.lcncCfg.Spec.Properties.Vars)
	}

	// process functions
	if fnc.lcncFunctionsPreHookFn != nil {
		fnc.lcncFunctionsPreHookFn(r.lcncCfg.Spec.Properties.Functions)
	}
	for idx, functions := range r.lcncCfg.Spec.Properties.Functions {
		// check if there is a block
		block := false
		if functions.ControllerConfigBlock.For != nil {
			block = true
			if fnc.lcncBlockFn != nil {
				fnc.lcncBlockFn(OriginFunction, idx, functions.ControllerConfigBlock)
			}
		}
		for vertexName, v := range functions.ControllerConfigFunctions {
			if fnc.lcncFunctionFn != nil {
				fnc.lcncFunctionFn(OriginFunction, block, idx, vertexName, v)
			}

		}
	}
	if fnc.lcncFunctionsPostHookFn != nil {
		fnc.lcncFunctionsPostHookFn(r.lcncCfg.Spec.Properties.Functions)
	}

	// process services
	if fnc.lcncServicesPreHookFn != nil {
		fnc.lcncServicesPreHookFn(r.lcncCfg.Spec.Properties.Services)
	}
	for idx, services := range r.lcncCfg.Spec.Properties.Services {
		for vertexName, v := range services.ControllerConfigFunctions {
			if fnc.lcncServiceFn != nil {
				fnc.lcncServiceFn(OriginService, false, idx, vertexName, v)
			}
		}
	}
	if fnc.lcncServicesPostHookFn != nil {
		fnc.lcncServicesPostHookFn(r.lcncCfg.Spec.Properties.Services)
	}

	// process config end
	if fnc.lcncCfgPostHookFn != nil {
		fnc.lcncCfgPostHookFn(r.lcncCfg)
	}
}
