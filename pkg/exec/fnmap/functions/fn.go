package functions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/exec/executor"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
)

type initOutputFn func(numItems int)
type recordOutputFn func(any)
type getFinalResultFn func() (output.Output, error)

// type prepareInputFn func(fnconfig *ctrlcfgv1.Function) any
type runFn func(context.Context, map[string]any) (any, error)

type fnExecConfig struct {
	executeRange  bool
	executeSingle bool
	// execution functions
	//prepareInputFn prepareInputFn
	runFn runFn
	// result functions
	initOutputFn     initOutputFn
	recordOutputFn   recordOutputFn
	getFinalResultFn getFinalResultFn
}

func (fec *fnExecConfig) exec(ctx context.Context, fnconfig *ctrlcfgv1.Function, input map[string]any) (output.Output, error) {
	var items []*item
	var isRange bool
	var ok bool
	var err error
	if fnconfig.HasBlock() {
		if fnconfig.Block.Range != nil {
			items, err = runRange(fnconfig.Block.Range.Value, input)
			if err != nil {
				return nil, err
			}
			isRange = true
		}
		if fnconfig.Block.Condition != nil {
			if exp := fnconfig.Block.Condition.Expression; exp != "" {
				ok, err = runCondition(exp, input)
				if err != nil {
					return nil, err
				}
				if !ok {
					return output.New(), executor.ErrConditionFalse // error to be ignored, condition false, so we dont have to run
				}
			}
			if fnconfig.Block.Condition.Block.Range != nil {
				items, err = runRange(fnconfig.Block.Condition.Block.Range.Value, input)
				if err != nil {
					return nil, err
				}
				isRange = true
			}
		}
	}
	numItems := len(items)
	if numItems == 0 && isRange {
		fec.initOutputFn(0)
		return nil, nil // no entries in the range, so we are done
	}
	if numItems > 0 && isRange {
		fec.initOutputFn(numItems)
		for i, item := range items {
			// this is a protection to ensure we dont use the nil result in a range
			if item.val != nil {
				input["VALUE"] = item.val
				input["KEY"] = fmt.Sprint(i)
				input["INDEX"] = i

				// resolve the local vars using jq and add them to the input
				if err := resolveLocalVars(fnconfig, input); err != nil {
					return nil, err
				}

				if fec.executeRange {
					//extraInput := fec.prepareInputFn(fnconfig)
					x, err := fec.runFn(ctx, input)
					if err != nil {
						return nil, err
					}
					fec.recordOutputFn(x)
				}
			}
		}
	}
	if fec.executeSingle {
		fec.initOutputFn(1)
		// resolve the local vars using jq and add them to the input
		if err := resolveLocalVars(fnconfig, input); err != nil {
			return nil, err
		}
		//extraInput := fec.prepareInputFn(fnconfig)
		x, err := fec.runFn(ctx, input)
		if err != nil {
			return nil, err
		}
		fec.recordOutputFn(x)
	}
	return fec.getFinalResultFn()
}

type item struct {
	//key string
	val any
}

func runRange(exp string, input map[string]any) ([]*item, error) {
	varNames := make([]string, 0, len(input))
	varValues := make([]any, 0, len(input))
	for name, v := range input {
		varNames = append(varNames, "$"+name)
		varValues = append(varValues, v)
	}
	fmt.Printf("runRange varNames: %v, varValues: %v\n", varNames, varValues)
	fmt.Printf("runRange exp: %s\n", exp)

	q, err := gojq.Parse(exp)
	if err != nil {
		return nil, err
	}
	code, err := gojq.Compile(q, gojq.WithVariables(varNames))
	if err != nil {
		return nil, err
	}
	result := make([]*item, 0)
	iter := code.Run(nil, varValues...)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if err != nil {
				fmt.Printf("runJQ err: %v\n", err)
				if strings.Contains(err.Error(), "cannot iterate over: null") {
					return result, nil
				}
				return nil, err
			}
		}
		if v == nil {
			continue
		}
		fmt.Printf("runRange result item: %#v\n", v)
		result = append(result, &item{val: v})
	}

	return result, nil
}

func runCondition(exp string, input map[string]any) (bool, error) {
	varNames := make([]string, 0, len(input))
	varValues := make([]any, 0, len(input))
	for name, v := range input {
		varNames = append(varNames, "$"+name)
		varValues = append(varValues, v)
	}
	fmt.Printf("runCondition varNames: %v, varValues: %v\n", varNames, varValues)
	fmt.Printf("runCondition exp: %s\n", exp)

	q, err := gojq.Parse(exp)
	if err != nil {
		return false, err
	}
	code, err := gojq.Compile(q, gojq.WithVariables(varNames))
	if err != nil {
		return false, err
	}
	iter := code.Run(nil, varValues...)

	v, ok := iter.Next()
	if !ok {
		return false, errors.New("not result")
	}

	if err, ok := v.(error); ok {
		if err != nil {
			fmt.Printf("runCondition err: %v\n", err)
			//if strings.Contains(err.Error(), "cannot iterate over: null") {
			//	return false, nil
			//}
			return false, err
		}
	}
	fmt.Printf("runCondition value: %t\n", v)
	if r, ok := v.(bool); ok {
		return r, nil
	}
	return false, fmt.Errorf("unexpected result type, want bool got %T", v)
}

func resolveLocalVars(fnconfig *ctrlcfgv1.Function, input map[string]any) error {
	if fnconfig.Vars != nil {
		fmt.Printf("resolveLocalVars: input: %v\n", input)
		for varName, expression := range fnconfig.Vars {
			// We are lazy and provide all reference input to JQ
			// the below aproach could be a more optimal solution
			// but for now we keep it simple

			//	localVarRefs := make(map[string]any)
			//	rfs := ccsyntax.NewReferences()
			//	refs := rfs.GetReferences(expression)
			//	for _, ref := range refs {
			//		localVarRefs[ref.Value] = input[ref.Value]
			//	}

			v, err := runJQ(expression, input)
			if err != nil {
				return err
			}
			fmt.Printf("resolveLocalVars jq %#v\n", v)
			/*
				b, err := yaml.Marshal(v)
				if err != nil {
					return err
				}
				x := map[string]interface{}
				if err:= yaml
			*/
			input[varName] = v
		}
	}
	return nil
}
