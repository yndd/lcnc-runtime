package fnruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/shlex"
	"github.com/henderiw-k8s-lcnc/fn-sdk/go/fn"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	fnresultv1 "github.com/yndd/lcnc-runtime/pkg/api/fnresult/v1"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnlib"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
)

type FunctionKind string

const (
	FuntionKindFunction FunctionKind = "function"
	FunctionKindService FunctionKind = "service"
)

type RunnerOptions struct {
	Kind FunctionKind

	// only used for kind =service, exposes the port used by the service in the container
	ServicePort int

	// ImagePullPolicy controls the image pulling behavior before running the container.
	ImagePullPolicy fnlib.ImagePullPolicy

	// allowExec determines if function binary executable are allowed
	// to be run during execution. Running function binaries is a
	// privileged operation, so explicit permission is required.
	AllowExec bool

	// allowWasm determines if function wasm are allowed to be run during
	// execution. Running wasm function is an alpha feature, so it needs to be
	// enabled explicitly.
	AllowWasm bool

	// ResolveToImage will resolve a partial image to a fully-qualified one
	ResolveToImage ImageResolveFunc
}

// ImageResolveFunc is the type for a function that can resolve a partial image to a (more) fully-qualified name
type ImageResolveFunc func(ctx context.Context, image string) (string, error)

func (o *RunnerOptions) InitDefaults() {
	o.ImagePullPolicy = fnlib.IfNotPresentPull
	o.ResolveToImage = ResolveToImageForCLI
}

type Runner interface {
	Run(ctx context.Context, rCtx *fn.ResourceContext) (*fn.ResourceContext, error)
}

type runner struct {
	opts     RunnerOptions
	fnRunner FunctionRunner
}

type FunctionRunner interface {
	FnRun(ctx context.Context, reader io.Reader, writer io.Writer) error
	SvcRun(ctx context.Context) error
}

// NewRunner returns a FunctionRunner given a specification of a function
// and it's config.
func NewRunner(
	ctx context.Context,
	fnc ctrlcfgv1.Function,
	opts RunnerOptions,
) (Runner, error) {
	r := &runner{
		opts: opts,
	}
	if fnc.Executor.Image != "" {
		// resolve partial image
		img, err := opts.ResolveToImage(ctx, fnc.Executor.Image)
		if err != nil {
			return nil, err
		}
		fnc.Executor.Image = img
	}

	fnResult := &fnresultv1.Result{
		Image: fnc.Executor.Image,
	}

	switch {
	case fnc.Executor.Image != "":
		// TODO WASM
		switch opts.Kind {
		case FunctionKindService:
			servicePort := strconv.Itoa(r.opts.ServicePort)
			home := os.Getenv("HOME")
			fmt.Printf("home: %s\n", home)
			r.fnRunner = &ContainerFn{
				Image:           fnc.Executor.Image,
				ImagePullPolicy: opts.ImagePullPolicy,
				FnResult:        fnResult,
				Perm: ContainerFnPermission{
					AllowNetwork: true,
					AllowMount:   true,
				},
				StorageMounts: []runtimeutil.StorageMount{
					{MountType: "bind", Src: filepath.Join(home, ".kube", "config"), DstPath: "/config"},
				},
				Env: []string{
					strings.Join([]string{"FN_SERVICE_PORT", servicePort}, "="),
					strings.Join([]string{"KUBECONFIG", "/config"}, "="),
				},
			}
		default:
			r.fnRunner = &ContainerFn{
				Image:           fnc.Executor.Image,
				ImagePullPolicy: opts.ImagePullPolicy,
				FnResult:        fnResult,
			}
		}
	case fnc.Executor.Exec != "":
		if opts.Kind == FunctionKindService {
			return nil, fmt.Errorf("service not supported with exec")
		}
		// TODO WASM
		var execArgs []string
		// assuming exec here
		s, err := shlex.Split(fnc.Executor.Exec)
		if err != nil {
			return nil, fmt.Errorf("exec command %q must be valid: %w", fnc.Executor.Exec, err)
		}
		execPath := fnc.Executor.Exec
		if len(s) > 0 {
			execPath = s[0]
		}
		if len(s) > 1 {
			execArgs = s[1:]
		}
		r.fnRunner = &ExecFn{
			Path:     execPath,
			Args:     execArgs,
			FnResult: fnResult,
		}
		//}
	default:
		return nil, fmt.Errorf("must specify `exec` or `image` to execute a function")
	}

	return r, nil
}

func (r *runner) Run(ctx context.Context, rCtx *fn.ResourceContext) (*fn.ResourceContext, error) {
	switch r.opts.Kind {
	case FunctionKindService:
		err := r.fnRunner.SvcRun(ctx)
		return nil, err
	default:
		in := &bytes.Buffer{}
		out := &bytes.Buffer{}

		b, err := json.Marshal(rCtx)
		if err != nil {
			return nil, err
		}

		_, err = in.Write(b)
		if err != nil {
			return nil, err
		}

		fmt.Printf("rctx before fn Execution:\n%s\n", in.String())

		// call the specific implementation of run (container, exec or wasm)
		ex := r.fnRunner.FnRun(ctx, in, out)
		if ex != nil {
			return nil, fmt.Errorf("fn run failed: %s", ex.Error())
		}
		//fmt.Printf("rctx after fn execution:\n%v\n", out.String())

		newrCtx := &fn.ResourceContext{}
		if err := json.Unmarshal(out.Bytes(), newrCtx); err != nil {
			return nil, err
		}
		return newrCtx, nil
	}
}
