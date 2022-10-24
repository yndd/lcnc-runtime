package fnruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/shlex"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	fnresultv1 "github.com/yndd/lcnc-runtime/pkg/api/fnresult/v1"
	rctxv1 "github.com/yndd/lcnc-runtime/pkg/api/resourcecontext/v1"
)

type Run func(reader io.Reader, writer io.Writer) error

type RunnerOptions struct {
	// ImagePullPolicy controls the image pulling behavior before running the container.
	ImagePullPolicy ImagePullPolicy

	// allowExec determines if function binary executable are allowed
	// to be run during execution. Running function binaries is a
	// privileged operation, so explicit permission is required.
	AllowExec bool

	// allowWasm determines if function wasm are allowed to be run during
	// execution. Running wasm function is an alpha feature, so it needs to be
	// enabled explicitly.
	// AllowWasm bool

	// ResolveToImage will resolve a partial image to a fully-qualified one
	ResolveToImage ImageResolveFunc
}

// ImageResolveFunc is the type for a function that can resolve a partial image to a (more) fully-qualified name
type ImageResolveFunc func(ctx context.Context, image string) (string, error)

func (o *RunnerOptions) InitDefaults() {
	o.ImagePullPolicy = IfNotPresentPull
	o.ResolveToImage = ResolveToImageForCLI
}

// NewRunner returns a FunctionRunner given a specification of a function
// and it's config.
func NewRunner(
	ctx context.Context,
	fn *ctrlcfgv1.Function,
	//fnResults *fnresultv1.ResultList,
	opts RunnerOptions,
	//runtime fn.FunctionRuntime,
) (*FunctionRunner, error) {
	if fn.Image != "" {
		// resolve partial image
		img, err := opts.ResolveToImage(ctx, fn.Image)
		if err != nil {
			return nil, err
		}
		fn.Image = img
	}

	fnResult := &fnresultv1.Result{
		Image:    fn.Image,
		ExecPath: fn.Exec,
	}

	var run Run
	switch {
	case fn.Image != "":
		// If allowWasm is true, we will use wasm runtime for image field.
		/*
			if opts.AllowWasm {

					wFn, err := NewWasmFn(NewOciLoader(filepath.Join(os.TempDir(), "kpt-fn-wasm"), f.Image))
					if err != nil {
						return nil, err
					}
					fltr.Run = wFn.Run

				return nil, fmt.Errorf("wasm not yet supported")
			} else {
		*/
		cfn := &ContainerFn{
			Image:           fn.Image,
			ImagePullPolicy: opts.ImagePullPolicy,
			Ctx:             ctx,
			FnResult:        fnResult,
		}
		run = cfn.Run
		//}
	case fn.Exec != "":
		// If AllowWasm is true, we will use wasm runtime for exec field.
		/*
			if opts.AllowWasm {
					wFn, err := NewWasmFn(&FsLoader{Filename: f.Exec})
					if err != nil {
						return nil, err
					}
					fltr.Run = wFn.Run
				return nil, fmt.Errorf("wasm not yet supported")
			} else {
		*/
		var execArgs []string
		// assuming exec here
		s, err := shlex.Split(fn.Exec)
		if err != nil {
			return nil, fmt.Errorf("exec command %q must be valid: %w", fn.Exec, err)
		}
		execPath := fn.Exec
		if len(s) > 0 {
			execPath = s[0]
		}
		if len(s) > 1 {
			execArgs = s[1:]
		}
		eFn := &ExecFn{
			Path:     execPath,
			Args:     execArgs,
			FnResult: fnResult,
		}
		run = eFn.Run
		//}
	default:
		return nil, fmt.Errorf("must specify `exec` or `image` to execute a function")
	}

	return NewFunctionRunner(ctx, run, opts)
}

// NewFunctionRunner returns a FunctionRunner given a specification of a function
// and it's config.
func NewFunctionRunner(ctx context.Context,
	//fltr *runtimeutil.FunctionFilter,
	run Run,
	//pkgPath types.UniquePath,
	//fnResult *fnresultv1.Result,
	//fnResults *fnresultv1.ResultList,
	opts RunnerOptions) (*FunctionRunner, error) {

	// by default, the inner most runtimeutil.FunctionFilter scopes resources to the
	// directory specified by the functionConfig, kpt v1+ doesn't scope resources
	// during function execution, so marking the scope to global.
	// See https://github.com/GoogleContainerTools/kpt/issues/3230 for more details.
	return &FunctionRunner{
		ctx: ctx,
		//name: name,
		//pkgPath:   pkgPath,
		//filter:    fltr,
		run: run,
		//fnResult:  fnResult,
		//fnResults: fnResults,
		opts: opts,
	}, nil
}

// FunctionRunner wraps FunctionFilter and implements kio.Filter interface.
type FunctionRunner struct {
	ctx context.Context
	//name string
	//pkgPath          types.UniquePath
	//disableCLIOutput bool
	//filter    *runtimeutil.FunctionFilter
	run Run
	//fnResult  *fnresultv1.Result
	//fnResults *fnresultv1.ResultList
	opts RunnerOptions
}

func (fr *FunctionRunner) Run(rctx *rctxv1.ResourceContext) (*rctxv1.ResourceContext, error) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}

	b, err := json.Marshal(rctx)
	if err != nil {
		return nil, err
	}

	_, err = in.Write(b)
	if err != nil {
		return nil, err
	}

	fmt.Printf("run rctx after printer: %v\n", in.String())

	// here we call the run
	ex := fr.run(in, out)
	if ex != nil {
		return nil, fmt.Errorf("fn run failed: %s", ex.Error())
	}

	newRctx := &rctxv1.ResourceContext{}
	if err := json.Unmarshal(out.Bytes(), newRctx); err != nil {
		return nil, err
	}
	//fmt.Printf("!!! rcctx: %+v\n", newRctx)
	return newRctx, nil
}
