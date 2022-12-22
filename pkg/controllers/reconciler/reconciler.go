package reconciler

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	"github.com/yndd/lcnc-runtime/pkg/ccsyntax"
	"github.com/yndd/lcnc-runtime/pkg/executor"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	"github.com/yndd/ndd-runtime/pkg/logging"
)

const (
	// errors
	errGetCr        = "cannot get resource"
	errUpdateStatus = "cannot update resource status"
	errMarshalCr    = "cannot marshal resource"

// reconcileFailed = "reconcile failed"
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
	}
}

type reconciler struct {
	client       client.Client
	pollInterval time.Duration
	ceCtx        ccsyntax.ConfigExecutionContext

	log logging.Logger
	//record event.Recorder
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")

	gvk := r.ceCtx.GetForGVK()
	//o := getUnstructured(r.gvk)
	cr := meta.GetUnstructuredFromGVK(gvk)
	if err := r.client.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the CR no longer exist we are done
		log.Debug(errGetCr, "error", err)
		return reconcile.Result{}, errors.Wrap(meta.IgnoreNotFound(err), errGetCr)
	}

	x, err := meta.MarshalData(cr)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, errMarshalCr)
	}

	/*
		if err := meta.AddFinalizer(cr, "finalizer string"); err != nil {
			log.Debug("Cannot add finalizer", "error", err)
			//managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
		}
	*/

	// delete branch -> used for delete
	if meta.WasDeleted(cr) {
		// handle delete branch
		deleteDAG := r.ceCtx.GetDAG(ccsyntax.FOWFor, ccsyntax.GVKOperation{GVK: *gvk, Operation: ccsyntax.OperationDelete})
		e := executor.New(&executor.Config{
			RootVertex: deleteDAG.GetRootVertex(),
			Data:       x,
			Client:     r.client,
			GVK:        gvk,
			DAG:        deleteDAG,
		})

		// TODO should be per crName
		e.Run(ctx, req)
		e.GetOutput()
		e.GetResult()

		// remove finalizer

		return reconcile.Result{}, nil
	}
	// apply branch -> used for create and update

	applyDAG := r.ceCtx.GetDAG(ccsyntax.FOWFor, ccsyntax.GVKOperation{GVK: *gvk, Operation: ccsyntax.OperationApply})
	e := executor.New(&executor.Config{
		RootVertex: applyDAG.GetRootVertex(),
		Data:       x,
		Client:     r.client,
		GVK:        gvk,
		DAG:        r.ceCtx.GetDAG(ccsyntax.FOWFor, ccsyntax.GVKOperation{GVK: *gvk, Operation: ccsyntax.OperationApply}),
	})

	// TODO should be per crName
	e.Run(ctx, req)
	e.GetOutput()
	e.GetResult()

	//time.Sleep(60 * time.Second)

	return reconcile.Result{}, nil
	//return reconcile.Result{RequeueAfter: r.pollInterval}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
}
