package fnmap

import (
	"context"
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

func (r *fnmap) RunFn(ctx context.Context, fnconfig *ctrlcfgv1.ControllerConfigFunction, input map[string]any) (any, error) {
	switch fnconfig.Type {
	case ctrlcfgv1.ForQueryType:
		return r.forQuery(ctx, input)
	case ctrlcfgv1.QueryType:
		return r.query(ctx, fnconfig, input)
	case ctrlcfgv1.MapType:
		if fnconfig.HasBlock() {
			var items []*item
			var isRange bool
			var ok bool
			var err error
			if fnconfig.Block.Range != nil {
				items, err = runRange(fnconfig.Block.Range.Value, input)
				if err != nil {
					return nil, err
				}
				isRange = true
			}
			if exp := fnconfig.Block.Condition.Expression; exp != "" {
				ok, err = runCondition(exp, input)
				if err != nil {
					return nil, err
				}
				if !ok {
					return nil, errors.New("does not need to run") // error to be ignored
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
		if fnconfig.Block.Range != nil {
			items, err = runRange(fnconfig.Block.Range.Value, input)
			if err != nil {
				return nil, err
			}
			isRange = true
		}
		if exp := fnconfig.Block.Condition.Expression; exp != "" {
			ok, err = runCondition(exp, input)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, errors.New("does not need to run") // error to be ignored
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

func runRange(exp string, input any) ([]*item, error) {
	q, err := gojq.Parse(exp)
	if err != nil {
		return nil, err
	}
	code, err := gojq.Compile(q)
	if err != nil {
		return nil, err
	}
	result := make([]*item, 0)
	iter := code.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		result = append(result, &item{val: v})
	}
	return result, nil
}

func runCondition(exp string, input any) (bool, error) {
	q, err := gojq.Parse(exp)
	if err != nil {
		return false, err
	}
	code, err := gojq.Compile(q)
	if err != nil {
		return false, err
	}
	iter := code.Run(input)

	v, ok := iter.Next()
	if !ok {
		return false, errors.New("not result")
	}

	if r, ok := v.(bool); ok {
		return r, nil
	}
	return false, fmt.Errorf("unexpected result type, want bool got %T", v)
}
