package fnmap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (r *fnmap) RunFn(ctx context.Context, fnconfig *ctrlcfgv1.Function, input map[string]any) (any, error) {
	switch fnconfig.Type {
	case ctrlcfgv1.ForQueryType:
		return r.forQuery(ctx, input)
	case ctrlcfgv1.QueryType:
		if fnconfig.HasBlock() {
			if fnconfig.Block.Condition != nil {
				if exp := fnconfig.Block.Condition.Expression; exp != "" {
					ok, err := runCondition(exp, input)
					if err != nil {
						return nil, err
					}
					if !ok {
						return nil, errors.New("does not need to run") // error to be ignored
					}
				}
			}
		}
		return r.query(ctx, fnconfig, input)
	case ctrlcfgv1.MapType:
		if fnconfig.HasBlock() {
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
							return nil, errors.New("does not need to run") // error to be ignored
						}
					}
				}
			}
			numItems := len(items)
			if numItems == 0 && isRange {
				return nil, nil
			}
			if numItems > 0 && isRange {
				result := make(map[string]any, numItems)
				for i, item := range items {
					varItems := []varItem{
						{name: "VALUE", value: item.val},
						{name: "KEY", value: fmt.Sprint(i)},
						{name: "INDEX", value: i},
					}
					k, v, err := buildKV(fnconfig.Input.Key, fnconfig.Input.Value, input, varItems...)
					if err != nil {
						return nil, err
					}
					result[k] = v
				}
				return result, nil
			}
			// TODO: run single function ?
		}
	case ctrlcfgv1.SliceType:
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
						return nil, errors.New("does not need to run") // error to be ignored
					}
				}
			}
		}

		numItems := len(items)
		if numItems == 0 && isRange {
			return nil, nil
		}
		if numItems > 0 && isRange {
			result := make([]any, 0, numItems)
			for i, item := range items {
				varItems := []varItem{
					{name: "VALUE", value: item.val},
					{name: "KEY", value: fmt.Sprint(i)},
					{name: "INDEX", value: i},
				}

				v, err := buildSliceItem(fnconfig.Input.Value, input, varItems...)
				if err != nil {
					return nil, err
				}
				result = append(result, v)
			}
			return result, nil
		}
	case ctrlcfgv1.JQType:
		return runJQ(fnconfig.Input.Expression, input)
	case "": // image
	}

	return nil, nil
}

type item struct {
	key string
	val any
}

func runRange(exp string, input map[string]any) ([]*item, error) {
	varNames := make([]string, 0, len(input))
	varValues := make([]any, 0, len(input))
	for name, v := range input {
		varNames = append(varNames, "$"+name)

		switch x := v.(type) {
		case unstructured.Unstructured:
			b, err := json.Marshal(x.UnstructuredContent())
			if err != nil {
				return nil, err
			}
			rj := map[string]interface{}{}
			if err := json.Unmarshal(b, &rj); err != nil {
				return nil, err
			}
			varValues = append(varValues, rj)
		default:
			varValues = append(varValues, v)
		}
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
		fmt.Printf("runRange result item: %v\n", v)
		result = append(result, &item{val: v})
	}

	return result, nil
}

func runCondition(exp string, input map[string]any) (bool, error) {
	varNames := make([]string, 0, len(input))
	varValues := make([]any, 0, len(input))
	for name, v := range input {
		varNames = append(varNames, "$"+name)

		switch x := v.(type) {
		case unstructured.Unstructured:
			b, err := json.Marshal(x.UnstructuredContent())
			if err != nil {
				return false, err
			}

			rj := map[string]interface{}{}
			if err := json.Unmarshal(b, &rj); err != nil {
				return false, err
			}
			varValues = append(varValues, rj)
		default:
			varValues = append(varValues, v)
		}
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
	iter := code.Run(nil, varValues)

	v, ok := iter.Next()
	if !ok {
		return false, errors.New("not result")
	}

	if err, ok := v.(error); ok {
		if err != nil {
			fmt.Printf("runCondition err: %v\n", err)
			if strings.Contains(err.Error(), "cannot iterate over: null") {
				return false, nil
			}
			return false, err
		}
	}

	if r, ok := v.(bool); ok {
		return r, nil
	}
	return false, fmt.Errorf("unexpected result type, want bool got %T", v)
}
