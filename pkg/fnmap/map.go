package fnmap

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func buildKV(key, value string, input map[string]any, vars ...varItem) (string, any, error) {
	if value == "" {
		return "", nil, errors.New("missing input value")
	}
	if key == "" {
		return "", nil, errors.New("missing input key")
	}
	varNames := make([]string, 0, len(vars)+len(input))
	varValues := make([]any, 0, len(vars)+len(input))
	for _, v := range vars {
		varNames = append(varNames, "$"+v.name)
		varValues = append(varValues, v.value)
	}
	for name, v := range input {
		varNames = append(varNames, "$"+name)

		switch x := v.(type) {
		case unstructured.Unstructured:
			b, err := json.Marshal(x.UnstructuredContent())
			if err != nil {
				return "", nil, err
			}

			rj := map[string]interface{}{}
			if err := json.Unmarshal(b, &rj); err != nil {
				return "", nil, err
			}
			varValues = append(varValues, rj)
		case []unstructured.Unstructured:
			rj := make([]interface{}, len(x))
			for _, v := range x {
				b, err := json.Marshal(v.UnstructuredContent())
				if err != nil {
					return "", nil, err
				}

				vrj := map[string]interface{}{}
				if err := json.Unmarshal(b, &vrj); err != nil {
					return "", nil, err
				}
				rj = append(rj, vrj)
			}
			varValues = append(varValues, rj)
		default:
			varValues = append(varValues, v)
		}
	}
	fmt.Printf("buildKV varNames: %v, varValues: %v\n", varNames, varValues)
	fmt.Printf("buildKV exp: %s\n", value)

	valq, err := gojq.Parse(value)
	if err != nil {
		fmt.Printf("buildKV valq: %s\n", err.Error())
		return "", nil, err
	}
	valC, err := gojq.Compile(valq, gojq.WithVariables(varNames))
	if err != nil {
		fmt.Printf("buildKV valC: %s\n", err.Error())
		return "", nil, err
	}
	keyq, err := gojq.Parse(key)
	if err != nil {
		fmt.Printf("buildKV keyq: %s\n", err.Error())
		return "", nil, err
	}
	keyC, err := gojq.Compile(keyq, gojq.WithVariables(varNames))
	if err != nil {
		fmt.Printf("buildKV keyC: %s\n", err.Error())
		return "", nil, err
	}

	v, err := runJQOnce(valC, nil, varValues...)
	if err != nil {
		fmt.Printf("buildKV runJQOnce valC: %s\n", err.Error())
		return "", nil, err
	}

	k, err := runJQOnce(keyC, nil, varValues...)
	if err != nil {
		fmt.Printf("buildKV runJQOnce keyC: %s\n", err.Error())
		return "", nil, err
	}
	fmt.Printf("buildKV k: %T %#v\n", k, k)
	ks, ok := k.(string)
	if !ok {
		return "", nil, fmt.Errorf("unexpected key format: %T", k)
	}
	return ks, v, nil
}
