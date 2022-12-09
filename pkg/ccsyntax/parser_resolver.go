package ccsyntax

import (
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
)

func (r *lcncparser) Resolve(d dag.DAG) []Result {
	rs := &rs{
		d:              d,
		result:         []Result{},
		localVariable:  map[string]interface{}{},
		output:         map[string]string{},
		rootVertexName: r.rootVertexName,
	}

	fnc := &WalkConfig{
		gvrObjectFn:        rs.addGvr,
		pipelineBlockFn:    rs.addBlock,
		functionFn:         rs.addFunction,
		pipelinePostHookFn: rs.resolvePipelinePostHookFn,
	}

	// walk the config to resolve the vars/functions/etc
	// we add the verteces in the graph and check for duplicate entries
	// we create an output mapping, which will be used in the 2nd step (edges/dependencies)
	// local variables are resolved within the function
	r.walkLcncConfig(fnc)
	// stop if errors were found
	if len(rs.result) != 0 {
		return rs.result
	}

	// the 2nd walk adds the dependencies and edges in the graph
	fnc = &WalkConfig{
		pipelinePostHookFn: rs.addDependenciesLcncFunctionsPostHookFn,
	}
	r.walkLcncConfig(fnc)

	return rs.result
}

type rs struct {
	rootVertexName string
	d              dag.DAG
	mr             sync.RWMutex
	result         []Result
	// key is the local variable
	ml            sync.RWMutex
	localVariable map[string]interface{}
	// key is the outputKey
	mo     sync.RWMutex
	output map[string]string
}

func (r *rs) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *rs) resolveGvrObjectFn(o Origin, idx int, vertexName string, v ctrlcfgv1.ControllerConfigGvrObject) {
	if err := r.d.AddVertex(vertexName, v); err != nil {
		r.recordResult(Result{
			Origin: o,
			Index:  idx,
			Name:   vertexName,
			Error:  err.Error(),
		})
	}
}

func (r *rs) resolveFunctionFn(o Origin, idx int, vertexName string, v ctrlcfgv1.ControllerConfigFunction) {
	if err := r.d.AddVertex(vertexName, v); err != nil {
		r.recordResult(Result{
			Origin: o,
			Index:  idx,
			Name:   vertexName,
			Error:  err.Error(),
		})
	}

	// if no output is defined the output key is the name of the function
	if v.Output == nil {
		if err := r.AddOutputMapping(vertexName, vertexName); err != nil {
			r.recordResult(Result{
				Origin: o,
				Index:  idx,
				Name:   vertexName,
				Error:  err.Error(),
			})
		}
	}
	for k := range v.Output {
		if err := r.AddOutputMapping(k, vertexName); err != nil {
			r.recordResult(Result{
				Origin: o,
				Index:  idx,
				Name:   vertexName,
				Error:  err.Error(),
			})
		}
	}
}

func (r *rs) resolvePipelinePostHookFn(v ctrlcfgv1.ControllerConfigPipeline) {
	unresolvedFns := r.resolveUnresolved(v.DeepCopy().Functions)
	if len(unresolvedFns) != 0 {
		for idxName := range unresolvedFns {
			vertexName, idx := ctrlcfgv1.GetIdxName(idxName)
			r.recordResult(Result{
				Origin: OriginFunction,
				Index:  idx,
				Name:   vertexName,
				Error:  fmt.Errorf("unresolved function").Error(),
			})
		}
	}
}

/*
func (r *rs) resolveUnresolvedVars(unresolved map[string]ctrlcfgv1.ControllerConfigVarBlock) map[string]ctrlcfgv1.ControllerConfigVarBlock {
	totalUnresolved := len(unresolved)
	for idxName, v := range unresolved {
		if r.resolveVariable(v) {
			delete(unresolved, idxName)
		}
	}
	// when the new unresolved is 0 we are done and all variabled are resolved
	newUnresolved := len(unresolved)
	if newUnresolved == 0 {
		return unresolved
	}
	if newUnresolved < totalUnresolved {
		r.resolveUnresolvedVars(unresolved)
	}
	return unresolved
}

func (r *rs) resolveVariable(v ctrlcfgv1.ControllerConfigVarBlock) bool {
	for vertexName, vv := range v.ControllerConfigVariables {
		forblock := false
		if v.For != nil && v.For.Range != nil {
			forblock = true
			if !r.isResolved(&OriginContext{Origin: OriginVariable, ForBlock: true}, *v.For.Range) {
				fmt.Printf("unresolved vertexName: %s\n", vertexName)
				return false
			}
		}
		if vv.Map != nil {
			if !r.resolveMap(&OriginContext{Origin: OriginVariable, ForBlock: forblock, Query: true}, vv.Map) {
				return false
			}
		}
		if vv.Slice != nil {
			if !r.resolveValue(&OriginContext{Origin: OriginVariable, ForBlock: forblock, Query: true}, vv.Slice.ControllerConfigValue) {
				return false
			}
		}
	}
	return true
}
*/

func (r *rs) resolveUnresolved(unresolvedFns map[string]ctrlcfgv1.ControllerConfigPipelineBlock) map[string]ctrlcfgv1.ControllerConfigPipelineBlock {
	totalUnresolvedFns := len(unresolvedFns)
	for idxName, v := range unresolvedFns {
		if r.resolveFunction(v) {
			delete(unresolvedFns, idxName)
		}
	}
	// when the new unresolved is 0 we are done and all variabled are resolved
	if len(unresolvedFns) == 0 {
		return unresolvedFns
	}
	if len(unresolvedFns) < totalUnresolvedFns {
		r.resolveUnresolved(unresolvedFns)
	}
	return unresolvedFns
}

func (r *rs) resolvePipeLineBlock(v ctrlcfgv1.ControllerConfigPipelineBlock) bool {
	if v.IsBlock() {

	}
	for vertexName, vv := range v.ControllerConfigFunctions {
		// initialize the local variables for local resolution
		r.initLocalVariables()
		forblock := false
		if v.For != nil && v.For.Range != nil {
			forblock = true
			if !r.isResolved(&OriginContext{Origin: OriginFunction, ForBlock: true}, *v.For.Range) {
				fmt.Printf("unresolved vertexName: %s\n", vertexName)
				return false
			}
		}
		for localVarName, v := range vv.Vars {
			// TODO how to handle this error better
			if err := r.AddLocalVariable(localVarName, v); err != nil {
				return false
			}
			if v.Map != nil {
				if !r.resolveMap(&OriginContext{Origin: OriginFunction, ForBlock: forblock, Query: true}, v.Map) {
					return false
				}
			}
			if v.Slice != nil {
				if !r.resolveValue(&OriginContext{Origin: OriginFunction, ForBlock: forblock, Query: true}, v.Slice.ControllerConfigValue) {
					return false
				}
			}
		}
		for _, v := range vv.Input {
			if !r.isResolved(&OriginContext{Origin: OriginFunction, ForBlock: forblock, Input: true}, v) {
				return false
			}
		}
	}
	return true
}

func (r *rs) resolveMap(o *OriginContext, v *ctrlcfgv1.ControllerConfigMap) bool {
	if v.Key != nil {
		if !r.isResolved(o, *v.Key) {
			return false
		}
	}
	if !r.resolveValue(o, v.ControllerConfigValue) {
		return false
	}
	return true
}

func (r *rs) resolveValue(o *OriginContext, v ctrlcfgv1.ControllerConfigValue) bool {
	if v.ControllerConfigQuery.Query != nil {
		if !r.isResolved(o, *v.ControllerConfigQuery.Query) {
			return false
		}
	}
	if v.String != nil {
		if !r.isResolved(o, *v.String) {
			return false
		}
	}
	return true
}

func (r *rs) isResolved(o *OriginContext, s string) bool {
	// we dont handle the validation here, since is handled before
	value, err := GetValue(s)
	if err != nil {
		// this should never happen since validation should have happened before
		return false
	}
	resolved := false
	switch value.Kind {
	case GVRKind:
		// resolution is global, so the only resolution we can validate is if the resource exists
		// on the api server
		resolved = true
	case ChildVariableReferenceKind, RootVariableReferenceKind:
		// input of a function can resolve to a local variable
		// if so we should be ok and dont have to add an edge since the variable has already been
		// resolved to handle the dependency
		if o.Origin == OriginFunction && o.Input && r.GetLocalVariable(value.Variable[0]) {
			resolved = true
			break
		}
		// a fucntion can be dependent on another fn based on the output
		if o.Origin == OriginFunction && r.HasOutputMapping(value.Variable[0]) {
			resolved = true
			break
		}
		// check if the global variable/output exists
		if r.d.GetVertex(value.Variable[0]) {
			resolved = true
			break
		}
	case KeyVariableReferenceKind:
		if o.ForBlock {
			resolved = true
			break
		}
	}
	//fmt.Printf("isResolved originContext: %v string: %s resolved: %t\n", *o, s, resolved)
	return resolved
}
