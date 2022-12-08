package ccsyntax

import (
	"fmt"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

func (r *rs) addDependenciesLcncVarsPostHookFn(vv []ctrlcfgv1.ControllerConfigVarBlock) {
	newvars := ctrlcfgv1.CopyVariables(vv)
	for idxName, v := range newvars {
		vertexName, idx := ctrlcfgv1.GetIdxName(idxName)

		if v.For != nil && v.For.Range != nil {
			if err := r.connectEdges(&OriginContext{
				Origin: OriginVariable,
			}, vertexName, *v.For.Range); err != nil {
				r.recordResult(Result{
					Origin: OriginVariable,
					Index:  idx,
					Name:   vertexName,
					Error:  err.Error(),
				})
			}
		}
		for _, v := range v.ControllerConfigVariables {
			if v.Map != nil {
				if err := r.connectEdgesMap(&OriginContext{
					Origin: OriginVariable,
				}, vertexName, v.Map); err != nil {
					r.recordResult(Result{
						Origin: OriginVariable,
						Index:  idx,
						Name:   vertexName,
						Error:  err.Error(),
					})
				}
			}
			if v.Slice != nil {
				if err := r.connectEdgesValue(&OriginContext{
					Origin: OriginVariable,
				}, vertexName, v.Slice.ControllerConfigValue); err != nil {
					r.recordResult(Result{
						Origin: OriginVariable,
						Index:  idx,
						Name:   vertexName,
						Error:  err.Error(),
					})
				}
			}
		}
	}
}

func (r *rs) addDependenciesLcncFunctionsPostHookFn(v []ctrlcfgv1.ControllerConfigFunctionsBlock) {
	r.initLocalVariables()
	newfns := ctrlcfgv1.CopyFunctions(v)
	for idxName, vv := range newfns {
		vertexName, idx := ctrlcfgv1.GetIdxName(idxName)
		forblock := false
		for _, v := range vv.ControllerConfigFunctions {
			if vv.For != nil && vv.For.Range != nil {
				forblock = true
				if err := r.connectEdges(&OriginContext{
					Origin:   OriginFunction,
					ForBlock: forblock,
				}, vertexName, *vv.For.Range); err != nil {
					r.recordResult(Result{
						Origin: OriginFunction,
						Index:  idx,
						Name:   vertexName,
						Error:  err.Error(),
					})
				}
			}
			for localVarName, v := range v.Vars {
				// TODO how to handle this error better
				if err := r.AddLocalVariable(localVarName, v); err != nil {
					r.recordResult(Result{
						Origin: OriginFunction,
						Index:  idx,
						Name:   vertexName,
						Error:  err.Error(),
					})
				}
				if v.Map != nil {
					if err := r.connectEdgesMap(&OriginContext{
						Origin:   OriginFunction,
						ForBlock: forblock,
					}, vertexName, v.Map); err != nil {
						r.recordResult(Result{
							Origin: OriginFunction,
							Index:  idx,
							Name:   vertexName,
							Error:  err.Error(),
						})
					}
				}
				if v.Slice != nil {
					if err := r.connectEdgesValue(&OriginContext{
						Origin:   OriginFunction,
						ForBlock: forblock,
					}, vertexName, v.Slice.ControllerConfigValue); err != nil {
						r.recordResult(Result{
							Origin: OriginFunction,
							Index:  idx,
							Name:   vertexName,
							Error:  err.Error(),
						})
					}
				}
			}
			for _, v := range v.Input {
				if err := r.connectEdges(&OriginContext{
					Origin:   OriginFunction,
					ForBlock: forblock,
					Input:    true,
				}, vertexName, v); err != nil {
					r.recordResult(Result{
						Origin: OriginFunction,
						Index:  idx,
						Name:   vertexName,
						Error:  err.Error(),
					})
				}
			}
		}
	}

}

func (r *rs) connectEdgesMap(o *OriginContext, vertexName string, v *ctrlcfgv1.ControllerConfigMap) error {
	if v.Key != nil {
		if err := r.connectEdges(o, vertexName, *v.Key); err != nil {
			return err
		}
	}
	if err := r.connectEdgesValue(o, vertexName, v.ControllerConfigValue); err != nil {
		return err
	}
	return nil
}

func (r *rs) connectEdgesValue(o *OriginContext, vertexName string, v ctrlcfgv1.ControllerConfigValue) error {
	if v.ControllerConfigQuery.Query != nil {
		if err := r.connectEdges(o, vertexName, *v.ControllerConfigQuery.Query); err != nil {
			return err
		}
	}
	if v.String != nil {
		if err := r.connectEdges(o, vertexName, *v.String); err != nil {
			return err
		}
	}
	return nil
}

func (r *rs) connectEdges(o *OriginContext, vertexName, s string) error {
	value, err := GetValue(s)
	if err != nil {
		return err
	}
	switch value.Kind {
	case GVRKind:
		r.d.Connect(r.rootVertexName, vertexName)
	case ChildVariableReferenceKind, RootVariableReferenceKind:
		// input of a function can resolve to a local variable
		// if so we should be ok and dont have to add an edge since the variable has already been
		// resolved to handle the dependency
		if o.Origin == OriginFunction && o.Input && r.GetLocalVariable(value.Variable[0]) {
			break
		}
		// a fucntion can be dependent on another fn based on the output
		if o.Origin == OriginFunction && r.HasOutputMapping(value.Variable[0]) {
			//fmt.Printf("connect with output %s -> %s, originContext: %v\n", r.GetOutputMapping(value.Variable[0]), vertexName, *o)
			//r.PrintOutputMappings()
			r.d.Connect(r.GetOutputMapping(value.Variable[0]), vertexName)
			break
		}
		//fmt.Printf("connect %s -> %s, originContext: %v\n", value.Variable[0], vertexName, *o)
		r.d.Connect(value.Variable[0], vertexName)
	case KeyVariableReferenceKind:
	default:
		return fmt.Errorf("cannot add edge: from %s, to: %s", value.Variable[0], vertexName)
	}
	return nil
}
