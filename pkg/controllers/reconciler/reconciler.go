package reconciler

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/yndd/lcnc-runtime/pkg/ccsyntax"
	"github.com/yndd/lcnc-runtime/pkg/executor"
	"github.com/yndd/ndd-runtime/pkg/logging"
)

const (
	// errors
	//errGetCr        = "cannot get resource"
	//errUpdateStatus = "cannot update status"

	//reconcileFailed = "reconcile failed"
)

type ReconcileInfo struct {
	Client       client.Client
	PollInterval time.Duration
	CeCtx        ccsyntax.ConfigExecutionContext

	Log logging.Logger
}

func New(ri *ReconcileInfo) reconcile.Reconciler {
	return &reconciler{
		client:       ri.Client,
		pollInterval: ri.PollInterval,
		ceCtx:        ri.CeCtx,
		log:          ri.Log,
		exec: executor.New(&executor.Config{
			Client: ri.Client,
			GVK:    ri.CeCtx.GetForGVK(),
			DAG:    ri.CeCtx.GetDAG(ccsyntax.FOWFor, ri.CeCtx.GetForGVK()),
		}),
	}
}

type reconciler struct {
	client       client.Client
	pollInterval time.Duration
	ceCtx        ccsyntax.ConfigExecutionContext
	exec         executor.Executor

	log logging.Logger
	//record event.Recorder
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")

	e := executor.New(&executor.Config{
		Client: r.client,
		GVK:    r.ceCtx.GetForGVK(),
		DAG:    r.ceCtx.GetDAG(ccsyntax.FOWFor, r.ceCtx.GetForGVK()),
	})

	// TODO should be per crName
	e.Run(ctx, req)
	e.GetOutput()
	e.GetResult()

	time.Sleep(60 * time.Second)

	return reconcile.Result{}, nil
	//return reconcile.Result{RequeueAfter: r.pollInterval}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
}
