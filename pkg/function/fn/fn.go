package fn

import (
	"context"
	"io"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

type FunctionRunner interface {
	// Run method accepts a resourceContext in wireformat and return resourceContext in wireformat
	Run(r io.Reader, w io.Writer) error
}

// FunctionRuntime provides a way to obtain a fucntion runner to be used for a
// given function config
type FunctionRuntime interface {
	GetRunner(ctx context.Context, fn *ctrlcfgv1.Function) (ServiceFunctionRunner, error)
}

// ServiceFunctionRunner knows how to run a service funntion
type ServiceFunctionRunner interface {
	// Run method runs a service function
	Run() error
}

// ServiceFunctionRuntime provides a way to obtain a service fucntion runner to be used for a
// given function config
type ServiceFunctionRuntime interface {
	GetRunner(ctx context.Context, fn *ctrlcfgv1.Function) (ServiceFunctionRunner, error)
}
