package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"github.com/henderiw-k8s-lcnc/fn-svc-sdk/pkg/svcclient"
	"github.com/pkg/errors"
	"github.com/yndd/lcnc-runtime/pkg/applicator"
	"github.com/yndd/lcnc-runtime/pkg/ccsyntax"
	"github.com/yndd/lcnc-runtime/pkg/event"
	"github.com/yndd/lcnc-runtime/pkg/exec/builder"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnmap"
	"github.com/yndd/lcnc-runtime/pkg/exec/output"
	"github.com/yndd/lcnc-runtime/pkg/exec/result"
	"github.com/yndd/lcnc-runtime/pkg/meta"
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
}

func New(c *Config) reconcile.Reconciler {
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	return &reconciler{
		client:       applicator.ClientApplicator{Client: c.Client, Applicator: applicator.NewAPIPatchingApplicator(c.Client)},
		pollInterval: c.PollInterval,
		ceCtx:        c.CeCtx,
		fnMap:        c.FnMap,
		l:            ctrl.Log.WithName("lcnc reconcile"),
		f:            meta.NewAPIFinalizer(c.Client, defaultFinalizerName),
		record:       event.NewNopRecorder(),
	}
}

type reconciler struct {
	client       applicator.ClientApplicator
	pollInterval time.Duration
	ceCtx        ccsyntax.ConfigExecutionContext
	fnMap        fnmap.FuncMap
	f            meta.Finalizer
	l            logr.Logger
	record       event.Recorder
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx)
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
		r.l.Error(err, "cannot marshal data")
		return reconcile.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
	}

	if err := r.f.AddFinalizer(ctx, cr); err != nil {
		r.l.Error(err, "cannot add finalizer")
		//managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
		return reconcile.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
	}

	sc, err := r.getSvcClients()
	if err != nil {
		r.l.Error(err, "get svc clients")
		return reconcile.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
	}
	for _, c := range sc {
		defer c.Close()
	}

	// delete branch -> used for delete
	if meta.WasDeleted(cr) {
		r.l.Info("reconcile delete started...")
		// handle delete branch
		deleteDAGCtx := r.ceCtx.GetDAGCtx(ccsyntax.FOWFor, gvk, ccsyntax.OperationDelete)

		o := output.New()
		result := result.New()
		e := builder.New(&builder.Config{
			Name:           req.Name,
			Namespace:      req.Namespace,
			Data:           x,
			Client:         r.client,
			GVK:            gvk,
			DAG:            deleteDAGCtx.DAG,
			Output:         o,
			Result:         result,
			ServiceClients: sc,
		})

		// TODO should be per crName
		e.Run(ctx)
		//o.Print()
		result.Print()

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

	o := output.New()
	result := result.New()
	e := builder.New(&builder.Config{
		Name:           req.Name,
		Namespace:      req.Namespace,
		Data:           x,
		Client:         r.client,
		GVK:            gvk,
		DAG:            applyDAGCtx.DAG,
		Output:         o,
		Result:         result,
		ServiceClients: sc,
	})

	e.Run(ctx)
	//o.Print()
	result.Print()

	// TODO check result if failed, return an error

	for _, output := range o.GetFinalOutput() {
		b, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			r.l.Error(err, "cannot marshal the content")
			return reconcile.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
		}
		//r.l.Info("final output", "jsin string", string(b))
		u := &unstructured.Unstructured{}
		if err := json.Unmarshal(b, u); err != nil {
			r.l.Error(err, "cannot unmarshal the content")
			return reconcile.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
		}
		r.l.Info("final output", "unstructured", u)

		r.l.Info("gvk", "cr", cr.GroupVersionKind(), "u", u.GroupVersionKind())

		if u.GroupVersionKind() == cr.GroupVersionKind() {
			cr = u
		} else {
			if err := r.client.Apply(ctx, u); err != nil {
				r.l.Error(err, "cannot apply the content")
				return reconcile.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
			}
		}
	}

	r.l.Info("reconcile apply finsihed...")
	return reconcile.Result{}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
}

func (r *reconciler) getSvcClients() (map[schema.GroupVersionKind]svcclient.ServiceClient, error) {
	// get a service client for each service instance
	sc := map[schema.GroupVersionKind]svcclient.ServiceClient{}
	for gvk, svcCtx := range r.ceCtx.GetServices().Get() {
		svcClient, err := svcclient.New(&svcclient.Config{
			Address:  strings.Join([]string{"127.0.0.1", strconv.Itoa(svcCtx.Port)}, ":"),
			Insecure: true,
		})
		if err != nil {
			r.l.Error(err, "cannot create new client")
			return nil, err
		}
		r.l.Info("svc client create", "gvk", gvk.String())
		fmt.Printf("client create: client: %v\n", svcClient.Get())
		sc[gvk] = svcClient
	}
	return sc, nil
}
