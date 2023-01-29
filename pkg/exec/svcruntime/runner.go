package svcruntime

import (
	"context"

	"github.com/yndd/lcnc-runtime/pkg/exec/fnlib"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

type Run func() error

type RunnerOptions struct {
	// ImagePullPolicy controls the image pulling behavior before running the container.
	ImagePullPolicy fnlib.ImagePullPolicy

	// allowExec determines if function binary executable are allowed
	// to be run during execution. Running function binaries is a
	// privileged operation, so explicit permission is required.
	AllowExec bool
}

func (o *RunnerOptions) InitDefaults() {
	o.ImagePullPolicy = fnlib.IfNotPresentPull
}

func NewRunner(
	ctx context.Context,
	fnc *ctrlcfgv1.Function,
	opts RunnerOptions,
)
