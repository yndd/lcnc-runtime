package functions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/yndd/lcnc-runtime/pkg/exec/input"
)

func runJQ(exp string, i input.Input) (any, error) {
	if exp == "" {
		return nil, errors.New("missing input value")
	}
	varNames := make([]string, 0, i.Length())
	varValues := make([]any, 0, i.Length())
	for name, v := range i.Get() {
		varNames = append(varNames, "$"+name)
		varValues = append(varValues, v)
	}
	//fmt.Printf("runJQ varNames: %v, varValues: %#v\n", varNames, varValues)
	//fmt.Printf("runJQ exp: %s\n", exp)

	q, err := gojq.Parse(exp)
	if err != nil {
		fmt.Printf("runJQ err: %s\n", err.Error())
		return nil, err
	}
	code, err := gojq.Compile(q, gojq.WithVariables(varNames))
	if err != nil {
		fmt.Printf("runJQ err: %s\n", err.Error())
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
				//fmt.Printf("runJQ err: %v\n", err)
				if strings.Contains(err.Error(), "cannot iterate over: null") {
					return result, nil
				}
				return nil, err
			}
		}
		//fmt.Printf("runJQ result item: %v\n", v)
		result = append(result, v)
	}
	return result, nil
}

func runJQOnce(code *gojq.Code, input any, vars ...any) (any, error) {
	iter := code.Run(input, vars...)

	v, ok := iter.Next()
	if !ok {
		return nil, errors.New("no result")
	}
	if err, ok := v.(error); ok {
		if err != nil {
			//fmt.Printf("runJQOnce err: %v\n", err)
			return nil, err
		}
	}
	return v, nil
}
