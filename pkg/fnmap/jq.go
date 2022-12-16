package fnmap

import (
	"errors"

	"github.com/itchyny/gojq"
)

func runJQ(exp string, input any, vars ...varItem) (any, error) {
	if exp == "" {
		return nil, errors.New("missing input value")
	}
	q, err := gojq.Parse(exp)
	if err != nil {
		return nil, err
	}
	varNames := make([]string, 0, len(vars))
	varValues := make([]any, 0, len(vars))
	for _, v := range vars {
		varNames = append(varNames, v.name)
		varValues = append(varValues, v.value)
	}
	code, err := gojq.Compile(q, gojq.WithVariables(varNames))
	if err != nil {
		return nil, err
	}

	iter := code.Run(input, varValues...)
	result := make([]any, 0)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		result = append(result, v)
	}
	return result, nil
}
