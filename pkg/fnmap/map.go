package fnmap

import (
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
)

func buildKV(key, value string, input any, vars ...varItem) (string, any, error) {
	if value == "" {
		return "", nil, errors.New("missing input value")
	}
	if key == "" {
		return "", nil, errors.New("missing input key")
	}
	varNames := make([]string, 0, len(vars))
	varValues := make([]any, 0, len(vars))
	for _, v := range vars {
		varNames = append(varNames, v.name)
		varValues = append(varValues, v.value)
	}

	valq, err := gojq.Parse(value)
	if err != nil {
		return "", nil, err
	}
	valC, err := gojq.Compile(valq, gojq.WithVariables(varNames))
	if err != nil {
		return "", nil, err
	}
	keyq, err := gojq.Parse(key)
	if err != nil {
		return "", nil, err
	}
	keyC, err := gojq.Compile(keyq, gojq.WithVariables(varNames))
	if err != nil {
		return "", nil, err
	}

	v, err := runJQOnce(valC, input, varValues)
	if err != nil {
		return "", nil, err
	}

	k, err := runJQOnce(keyC, input, varValues)
	if err != nil {
		return "", nil, err
	}
	ks, ok := k.(string)
	if !ok {
		return "", nil, fmt.Errorf("unexpected key format: %T", v)
	}
	return ks, v, nil
}
