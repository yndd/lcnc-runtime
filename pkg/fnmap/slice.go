package fnmap

import (
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
	// ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

type varItem struct {
	name  string
	value any
}

func buildSliceItem(value string, input any, vars ...varItem) (any, error) {
	if value == "" {
		return nil, errors.New("missing input value")
	}
	q, err := gojq.Parse(value)
	if err != nil {
		return nil, err
	}
	varNames := make([]string, 0, len(vars))
	varValues := make([]any, 0, len(vars))
	for _, v := range vars {
		varNames = append(varNames, "$"+v.name)
		varValues = append(varValues, v.value)
	}
	fmt.Printf("buildSliceItem varNames: %v, varValues: %v\n", varNames, varValues)
	fmt.Printf("buildSliceItem exp: %s\n", value)

	code, err := gojq.Compile(q, gojq.WithVariables(varNames))
	if err != nil {
		return nil, err
	}

	iter := code.Run(input, varValues...)
	v, ok := iter.Next()
	if !ok {
		return nil, errors.New("no value")
	}
	if err, ok := v.(error); ok {
		if err != nil {
			fmt.Printf("buildSliceItem err: %v\n", err)
			return nil, err
		}
	}
	fmt.Printf("buildSliceItem value: %v\n", v)
	return v, nil
}

func runJQOnce(code *gojq.Code, input any, vars ...any) (any, error) {
	iter := code.Run(input, vars)

	v, ok := iter.Next()
	if !ok {
		return nil, errors.New("no result")
	}
	return v, nil
}
