package reconciler

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/yndd/lcnc-runtime/pkg/ccsyntax"
	"github.com/yndd/lcnc-runtime/pkg/exec/builder"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	"github.com/yndd/ndd-runtime/pkg/event"
	"github.com/yndd/ndd-runtime/pkg/logging"
)

const (
	// const
	defaultFinalizerName = "lcnc.yndd.io/finalizer"
	// errors
	errGetCr        = "cannot get resource"
	errUpdateStatus = "cannot update resource status"
	errMarshalCr    = "cannot marshal resource"

// reconcileFailed = "reconcile failed"

)

type Config struct {
	Client       client.Client
	PollInterval time.Duration
	CeCtx        ccsyntax.ConfigExecutionContext
	FnMap        fnmap.FuncMap

	Log logging.Logger
}

func New(c *Config) reconcile.Reconciler {
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	return &reconciler{
		client:       c.Client,
		pollInterval: c.PollInterval,
		ceCtx:        c.CeCtx,
		fnMap:        c.FnMap,
		l:            ctrl.Log.WithName("lcnc reconcile"),
		f:            meta.NewAPIFinalizer(c.Client, defaultFinalizerName),
		record:       event.NewNopRecorder(),
	}
}

type reconciler struct {
	client       client.Client
	pollInterval time.Duration
	ceCtx        ccsyntax.ConfigExecutionContext
	fnMap        fnmap.FuncMap
	f            meta.Finalizer
	l            logr.Logger
	record       event.Recorder
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	r.l.Info("reconcile start...")

	gvk := r.ceCtx.GetForGVK()
	//o := getUnstructured(r.gvk)
	cr := meta.GetUnstructuredFromGVK(gvk)
	if err := r.client.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the CR no longer exist we are done
		r.l.Info(errGetCr, "error", err)
		return reconcile.Result{}, errors.Wrap(meta.IgnoreNotFound(err), errGetCr)
	}

	//record := r.record.WithAnnotations()

	x, err := meta.MarshalData(cr)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, errMarshalCr)
	}

	if err := r.f.AddFinalizer(ctx, cr); err != nil {
		r.l.Error(err, "cannot add finalizer")
		//managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
	}

	// delete branch -> used for delete
	if meta.WasDeleted(cr) {
		r.l.Info("reconcile delete started...")
		// handle delete branch
		deleteDAGCtx := r.ceCtx.GetDAGCtx(ccsyntax.FOWFor, gvk, ccsyntax.OperationDelete)
		/*
			e := executor.New(&executor.Config{
				Name:       req.Name,
				Namespace:  req.Namespace,
				RootVertex: deleteDAGCtx.RootVertexName,
				Data:       x,
				Client:     r.client,
				GVK:        gvk,
				DAG:        deleteDAGCtx.DAG,
			})
		*/

		o := output.New()
		e := builder.New(&builder.Config{
			Name:           req.Name,
			Namespace:      req.Namespace,
			RootVertexName: deleteDAGCtx.RootVertexName,
			Data:           x,
			Client:         r.client,
			GVK:            gvk,
			DAG:            deleteDAGCtx.DAG,
		})

		// TODO should be per crName
		result := e.Run(ctx)
		o.PrintOutput()
		result.PrintResult()

		if err := r.f.RemoveFinalizer(ctx, cr); err != nil {
			r.l.Error(err, "cannot remove finalizer")
			//managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
		}

		r.l.Info("reconcile delete finished...")

		return reconcile.Result{}, nil
	}
	// apply branch -> used for create and update
	r.l.Info("reconcile apply started...")
	applyDAGCtx := r.ceCtx.GetDAGCtx(ccsyntax.FOWFor, gvk, ccsyntax.OperationApply)
	/*
		e := executor.New(&executor.Config{
			Name:       req.Name,
			Namespace:  req.Namespace,
			RootVertex: applyDAGCtx.RootVertexName,
			Data:       x,
			Client:     r.client,
			GVK:        gvk,
			DAG:        applyDAGCtx.DAG,
		})
	*/

	o := output.New()
	e := builder.New(&builder.Config{
		Name:           req.Name,
		Namespace:      req.Namespace,
		RootVertexName: applyDAGCtx.RootVertexName,
		Data:           x,
		Client:         r.client,
		GVK:            gvk,
		DAG:            applyDAGCtx.DAG,
		Output:         o,
	})

	// TODO should be per crName
	result := e.Run(ctx)
	o.PrintOutput()
	result.PrintResult()

	//time.Sleep(60 * time.Second)

	r.l.Info("reconcile apply finsihed...")

	return reconcile.Result{}, nil
	//return reconcile.Result{RequeueAfter: r.pollInterval}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
}
