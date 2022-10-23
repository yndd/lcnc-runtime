package reconciler

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/fnruntime"
	"github.com/yndd/ndd-runtime/pkg/logging"
	"github.com/yndd/ndd-runtime/pkg/resource"
)

const (
	// errors
	errGetCr        = "cannot get resource"
	errUpdateStatus = "cannot update status"

	//reconcileFailed = "reconcile failed"
)

type ReconcileInfo struct {
	Client       client.Client
	PollInterval time.Duration
	Gvk          schema.GroupVersionKind
	Fn           *ctrlcfgv1.Function

	Log logging.Logger
}

func New(ri *ReconcileInfo) reconcile.Reconciler {
	return &reconciler{
		client:       ri.Client,
		pollInterval: ri.PollInterval,
		gvk:          ri.Gvk,
		fn:           ri.Fn,
		log:          ri.Log,
	}
}

type reconciler struct {
	client       client.Client
	pollInterval time.Duration
	gvk          schema.GroupVersionKind
	fn           *ctrlcfgv1.Function

	log logging.Logger
	//record event.Recorder
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")

	cr := getUnstructuredObj(r.gvk)
	if err := r.client.Get(ctx, req.NamespacedName, cr); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise we'll be
		// requeued implicitly because we return an error.
		log.Debug("Cannot get resource", "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetCr)
	}
	//log.Debug("get resource", "cr", cr.UnstructuredContent())

	// INJECT RUNNER
	if r.fn != nil {
		runner, err := fnruntime.NewRunner(
			ctx,
			r.fn,
			fnruntime.RunnerOptions{},
		)
		if err != nil {
			log.Debug("Cannot get runner", "error", err)
			return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetCr)
		}
		newCr, err := runner.Run(cr)
		if err != nil {
			log.Debug("run failed", "error", err)
			return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetCr)
		}
		log.Debug("cr after run", "cr", newCr.UnstructuredContent())
	}

	return reconcile.Result{RequeueAfter: r.pollInterval}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
}

func getUnstructuredObj(gvk schema.GroupVersionKind) *unstructured.Unstructured {
	var u unstructured.Unstructured
	u.SetAPIVersion(gvk.GroupVersion().String())
	u.SetKind(gvk.Kind)
	uCopy := u.DeepCopy()
	return uCopy
}
