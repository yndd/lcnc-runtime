package fnmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func runJQ(exp string, input map[string]any) (any, error) {
	if exp == "" {
		return nil, errors.New("missing input value")
	}
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
		}
	}
	fmt.Printf("runJQ varNames: %v, varValues: %v\n", varNames, varValues)
	fmt.Printf("runJQ exp: %s\n", exp)

	q, err := gojq.Parse(exp)
	if err != nil {
		return nil, err
	}
	code, err := gojq.Compile(q, gojq.WithVariables(varNames))
	if err != nil {
		return nil, err
	}

	result := make([]any, 0)
	iter := code.Run(nil, varValues...)
	for {
		v, ok := iter.Next()
		if !ok { // should this not be later
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
		fmt.Printf("runJQ result item: %v\n", v)
		result = append(result, v)
	}
	return result, nil
}
