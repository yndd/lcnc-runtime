package executor

import "errors"

var (
	ErrConditionFalse = errors.New("condition false, no need to run")
)
