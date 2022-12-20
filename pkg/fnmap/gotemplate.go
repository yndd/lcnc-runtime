package fnmap

import (
	"bytes"
	"errors"

	"text/template"
)

func runGT(tmpl string, input map[string]any) (any, error) {
	if tmpl == "" {
		return nil, errors.New("missing template")
	}
	result := new(bytes.Buffer)
	// TODO: add template custom functions
	tpl, err := template.New("default").Parse(tmpl)
	if err != nil {
		return nil, err
	}
	err = tpl.Execute(result, input)
	return result.String(), err
}
