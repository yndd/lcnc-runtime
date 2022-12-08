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
	rctxv1 "github.com/yndd/lcnc-runtime/pkg/api/resourcecontext/v1"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/ndd-runtime/pkg/logging"
	"github.com/yndd/ndd-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Root         string
	Dag          dag.DAG
	//Fn           *ctrlcfgv1.ControllerConfigFunction

	Log logging.Logger
}

func New(ri *ReconcileInfo) reconcile.Reconciler {
	return &reconciler{
		client:       ri.Client,
		pollInterval: ri.PollInterval,
		gvk:          ri.Gvk,
		//fn:           ri.Fn,
		log: ri.Log,
	}
}

type reconciler struct {
	client       client.Client
	pollInterval time.Duration
	gvk          schema.GroupVersionKind
	root         string
	d            dag.DAG
	//fn           *ctrlcfgv1.ControllerConfigFunction

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

	if err := r.client.List(ctx, getUnstructuredList(r.gvk)); err != nil {
		log.Debug("Cannot get resource", "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetCr)
	}
	//log.Debug("get resource", "cr", cr.UnstructuredContent())

	// INJECT RUNNER
	/*
		log.Debug("function", "fn name", r.fn)
		if r.fn != nil {
			runner, err := fnruntime.NewRunner(
				ctx,
				r.fn,
				fnruntime.RunnerOptions{
					ResolveToImage: fnruntime.ResolveToImageForCLI,
				},
			)
			if err != nil {
				log.Debug("cannot get runner", "error", err)
				return reconcile.Result{}, errors.Wrap(err, "cannot get runner")
			}

			rctx, err := buildResourceContext(cr)
			if err != nil {
				log.Debug("Cannot build resource context", "error", err)
				return reconcile.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(err, "cannot build resource context")
			}
			newRctx, err := runner.Run(rctx)
			if err != nil {
				log.Debug("run failed", "error", err)
				return reconcile.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(err, "run failed")
			}
			//log.Debug("cr after run", "cr", newRctx.Spec.Properties)
			b, err := json.MarshalIndent(newRctx, "", "  ")
			if err != nil {
				log.Debug("run failed", "error", err)
				return reconcile.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(err, "run failed")
			}
			fmt.Printf("new cr content:\n%s\n", string(b))
		}
	*/

	return reconcile.Result{RequeueAfter: r.pollInterval}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
}

func getUnstructuredList(gvk schema.GroupVersionKind) *unstructured.UnstructuredList {
	var u unstructured.UnstructuredList
	u.SetAPIVersion(gvk.GroupVersion().String())
	u.SetKind(gvk.Kind)
	uCopy := u.DeepCopy()
	return uCopy
}

func getUnstructuredObj(gvk schema.GroupVersionKind) *unstructured.Unstructured {
	var u unstructured.Unstructured
	u.SetAPIVersion(gvk.GroupVersion().String())
	u.SetKind(gvk.Kind)
	uCopy := u.DeepCopy()
	return uCopy
}

func buildResourceContext(cr *unstructured.Unstructured) (*rctxv1.ResourceContext, error) {
	inputCr, err := cr.MarshalJSON()
	if err != nil {
		return nil, err
	}

	rctx := &rctxv1.ResourceContext{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ResourceContext",
			APIVersion: "lcnc.yndd.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.GetName(),
			Namespace: cr.GetNamespace(),
		},
		Spec: rctxv1.ResourceContextSpec{
			Properties: &rctxv1.ResourceContextProperties{
				Input: map[string][]rctxv1.KRMResource{
					cr.GroupVersionKind().String(): {rctxv1.KRMResource(inputCr)},
				},
			},
		},
	}

	gvk := schema.GroupVersionKind{
		Group:   "lcnc.yndd.io",
		Version: "v1",
		Kind:    "ResourceContext",
	}

	rctx.SetGroupVersionKind(gvk)
	return rctx, nil

	//b := new(strings.Builder)
	//p := printers.JSONPrinter{}
	//p.PrintObj(rctx, b)
	//return []byte(b.String()), nil
}
