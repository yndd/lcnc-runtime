package ccsyntax

import (
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) ValidateSyntax() []Result {
	vs := &vs{
		result: []Result{},
	}

	fnc := &WalkConfig{
		cfgPreHookFn:           vs.validatePreHook,
		gvkObjectFn:            vs.validateGvk,
		emptyPipelineFn:        vs.validateEmptyPipeline,
		emptyFunctionElementFn: vs.validateEmptyFunctionElement,
		functionBlockFn:        vs.validateFunctionBlock,
		functionFn:             vs.validateFunction,
		serviceFn:              vs.validateServiceFunction,
	}

	// walk the config to validate the syntax
	r.walkLcncConfig(fnc)
	return vs.result

}

type vs struct {
	mr     sync.RWMutex
	result []Result
}

func (r *vs) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *vs) validatePreHook(lcncCfg *ctrlcfgv1.ControllerConfig) {
	if len(lcncCfg.Spec.Properties.For) != 1 {
		r.recordResult(Result{
			OriginContext: &OriginContext{FOWS: FOWFor},
			Error:         fmt.Errorf("lcnc config must have just 1 for statement, got: %v", lcncCfg.Spec.Properties.For).Error(),
		})
	}
}

func (r *vs) validateGvk(oc *OriginContext, v *ctrlcfgv1.GvkObject) *schema.GroupVersionKind {
	if len(v.Resource.Raw) == 0 {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("a gvk must be present, got: %v", v).Error(),
		})
	}
	gvk, err := ctrlcfgv1.GetGVK(v.Resource)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	return gvk
}

func (r *vs) validateEmptyPipeline(oc *OriginContext, v *ctrlcfgv1.GvkObject) {
	issue := false
	switch oc.FOWS {
	case FOWFor:
		// for a for we need to supply both an apply AND a delete pipeline
		issue = true
	case FOWWatch:
		// for a watch we need an apply pipeline but NOT a delete pipeline
		if oc.Operation == OperationApply {
			issue = true
		}
	default:
		// FOWOwn we dont need a pipeline
	}
	if issue {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("a %s pipeline must be present for %v", oc.Operation, *oc).Error(),
		})
	}
}

func (r *vs) validateEmptyFunctionElement(oc *OriginContext) {
	r.recordResult(Result{
		OriginContext: oc,
		Error:         fmt.Errorf("a function cannot be empty %v", *oc).Error(),
	})
}

// validateFunctionBlock validates the fucntion block
// if recursion is happening
// if there is a block
// if the block has the right symentics
func (r *vs) validateFunctionBlock(oc *OriginContext, v *ctrlcfgv1.FunctionElement) {
	if oc.BlockIndex >= 1 {
		// we can only have 1 block index -> only 1 recursion allowed
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("a pipeline van only have 1function block %v", *oc).Error(),
		})
	}
	if !v.Function.HasBlock() {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("a function block must have a block %v", *oc).Error(),
		})
	}
	if v.HasBlock() {
		r.validateBlock(oc, v.Block)
	}
}

// valdates the function
func (r *vs) validateFunction(oc *OriginContext, v *ctrlcfgv1.Function) {
	// validate block
	if v.HasBlock() {
		r.validateBlock(oc, v.Block)
	}

	// validate the function type
	switch v.Type {
	case ctrlcfgv1.MapType:
		if v.Input.Key == "" {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("key needs to be present in %s", v.Type).Error(),
			})
		}
		if v.Input.Value == "" {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("value needs to be present in %s", v.Type).Error(),
			})
		}
	case ctrlcfgv1.SliceType:
		if v.Input.Value == "" {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("value needs to be present in %s", v.Type).Error(),
			})
		}
	case ctrlcfgv1.QueryType:
		if v.Input != nil {
			if len(v.Input.Resource.Raw) == 0 {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         fmt.Errorf("gvk needs to be present in %s", v.Type).Error(),
				})
			} else {
				_, err := ctrlcfgv1.GetGVK(v.Input.Resource)
				if err != nil {
					r.recordResult(Result{
						OriginContext: oc,
						Error:         err.Error(),
					})
				}
			}
		} else {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("input needs to be present in %s", v.Type).Error(),
			})
		}

	case ctrlcfgv1.GoTemplateType:
		if len(v.Input.Resource.Raw) == 0 && v.Input.Template == "" {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("resource or template need to be present in %s", v.Type).Error(),
			})
		}
		if len(v.Input.Resource.Raw) != 0 {
			_, err := ctrlcfgv1.GetGVK(v.Input.Resource)
			if err != nil {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         err.Error(),
				})
			}
		}
	case ctrlcfgv1.BlockType:
		// nothing to do since this is already validated
	case ctrlcfgv1.ContainerType, ctrlcfgv1.WasmType:
		if v.Executor.Exec == "" && v.Executor.Image == "" {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("external functions need an image or exec, got %v", v).Error(),
			})
		}
	default:
	}

	// validate input references
	// e.g. check if a VALUE, KEY, INDEX is not used when no block is present
	if v.Input == nil {
		if !(v.Type == ctrlcfgv1.BlockType || v.Type == ctrlcfgv1.ContainerType) {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("input is needed in a function %s", v.Type).Error(),
			})
		}
	} else {
		if v.Input.Key != "" {
			r.validateContext(oc, v, v.Input.Key)
		}
		if v.Input.Value != "" {
			r.validateContext(oc, v, v.Input.Value)
		}
		for _, val := range v.Input.GenericInput {
			r.validateContext(oc, v, val)
		}
	}

	// validate Ouput
	// for external output a GVK needs to be present + validate the GVK syntax
	if v.Output != nil {
		for _, v := range v.Output {
			if len(v.Resource.Raw) == 0 {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         fmt.Errorf("cannot use output without data").Error(),
				})
			} else {
				_, err := ctrlcfgv1.GetGVK(v.Resource)
				if err != nil {
					r.recordResult(Result{
						OriginContext: oc,
						Error:         err.Error(),
					})
				}
			}
		}
	}

	// validate local vars -> TBD

}

// valdates the service function
func (r *vs) validateServiceFunction(oc *OriginContext, v *ctrlcfgv1.Function) {
	// validate Ouput
	if v.Type != ctrlcfgv1.ContainerType {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("cannot use services with type other than container").Error(),
		})
	}
	if v.Image == "" {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("cannot use services with an image").Error(),
		})
	}
	// for output a GVK needs to be present + validate the GVK syntax
	if v.Output != nil {
		for _, v := range v.Output {
			if len(v.Resource.Raw) == 0 {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         fmt.Errorf("cannot use output without data").Error(),
				})
			} else {
				_, err := ctrlcfgv1.GetGVK(v.Resource)
				if err != nil {
					r.recordResult(Result{
						OriginContext: oc,
						Error:         err.Error(),
					})
				}
			}
		}
	} else {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("cannot use a service w/o output definition").Error(),
		})
	}
}

func (r *vs) validateBlock(oc *OriginContext, v ctrlcfgv1.Block) {
	// process and validate block
	if v.Range != nil && v.Condition != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         fmt.Errorf("cannot have both range and condition in the same block, got: %v", v).Error(),
		})
	}
	if v.Range != nil {
		if v.Range.Value != "" {
			r.validateContext(oc, &ctrlcfgv1.Function{Block: v}, v.Range.Value)
		} else {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("range value cannot be empty: %v", v).Error(),
			})
		}
		if v.Range.Range != nil || v.Range.Condition != nil {
			r.validateBlock(oc, v.Range.Block)
		}
	}
	if v.Condition != nil {
		if v.Condition.Expression != "" {
			r.validateContext(oc, &ctrlcfgv1.Function{Block: v}, v.Condition.Expression)
		} else {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("condition expression cannot be empty: %v", v).Error(),
			})
		}
		if v.Condition.Range != nil || v.Condition.Condition != nil {
			r.validateBlock(oc, v.Condition.Block)
		}
	}
}

//func (r *vs) validateService(oc *OriginContext, v *ctrlcfgv1.ControllerConfigFunction) {
//}

func (r *vs) validateContext(oc *OriginContext, v *ctrlcfgv1.Function, s string) {
	rfs := NewReferences()
	refs := rfs.GetReferences(s)
	//fmt.Printf("validate ctxName: %s, value: %s, kind: %s, variable: %v\n", o.VertexName, s, value.Kind, value.Variable)
	for _, ref := range refs {
		switch ref.Kind {
		case RangeReferenceKind:
			if !v.HasBlock() || !v.Block.HasRange() {
				//fmt.Printf("validate ctx: vertex %s, ref: %s, string: %s, function value: %v\n", oc.VertexName, ref, s, v.Block)
				r.recordResult(Result{
					OriginContext: oc,
					Error:         fmt.Errorf("cannot use Key variables without a range statement").Error(),
				})
			}
			/*
				TOO RESTRICTIVE FOR NOW -> Need a more extensive validation policy
				if v.HasBlock() {
					r.recordResult(Result{
						OriginContext: oc,
						Error:         fmt.Errorf("cannot use Key variables in a block").Error(),
					})
				}
			*/
		case RegularReferenceKind:
			// this is ok
		default:
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("unknown reference kind, got: %s", s).Error(),
			})
		}
	}
}
