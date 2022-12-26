package executor

import "errors"

var (
	ErrNoKRM          = errors.New("object is not KRM")
	ErrConditionFalse = errors.New("condition false, no need to run")
)
