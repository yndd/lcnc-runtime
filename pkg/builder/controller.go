package builder

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/yndd/lcnc-runtime/pkg/ccsyntax"
	"github.com/yndd/lcnc-runtime/pkg/controller"
	"github.com/yndd/lcnc-runtime/pkg/manager"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var newController = controller.New

type Builder interface {
	Build(r reconcile.Reconciler) (controller.Controller, error)
}

type builder struct {
	ceCtx            ccsyntax.ConfigExecutionContext
	mgr              manager.Manager
	globalPredicates []predicate.Predicate
	ctrl             controller.Controller
	ctrlOptions      controller.Options
}

func New(mgr manager.Manager, ceCtx ccsyntax.ConfigExecutionContext, opts controller.Options) Builder {
	b := &builder{
		mgr:   mgr,
		ceCtx: ceCtx,
		ctrlOptions: opts,
	}
	return b
}

func (blder *builder) Build(r reconcile.Reconciler) (controller.Controller, error) {
	if blder.mgr == nil {
		return nil, fmt.Errorf("must provide a non-nil Manager")
	}
	if len(blder.ceCtx.GetFOW(ccsyntax.FOWFor)) != 1 {
		return nil, fmt.Errorf("cannot have more than 1 for")
	}
	// Set the ControllerManagedBy
	if err := blder.doController(r); err != nil {
		return nil, err
	}

	// Set the Watch
	if err := blder.doWatch(); err != nil {
		return nil, err
	}
	return blder.ctrl, nil
}

func (blder *builder) doWatch() error {
	// handle For
	// we validate that there is only 1 for so we are ok here

	gvk := blder.ceCtx.GetForGVK()
	typeForSrc := meta.GetUnstructuredFromGVK(gvk)
	src := &source.Kind{Type: typeForSrc}
	hdler := &handler.EnqueueRequestForObject{}
	allPredicates := append(blder.globalPredicates, []predicate.Predicate{}...)
	if err := blder.ctrl.Watch(src, hdler, allPredicates...); err != nil {
		return err
	}

	// hanlde Own
	// Watches the managed types
	for gvk := range blder.ceCtx.GetFOW(ccsyntax.FOWOwn) {
		obj := meta.GetUnstructuredFromGVK(gvk)

		src := &source.Kind{Type: obj}
		hdler := &handler.EnqueueRequestForOwner{
			OwnerType:    typeForSrc,
			IsController: true,
		}
		allPredicates := append([]predicate.Predicate(nil), blder.globalPredicates...)
		allPredicates = append(allPredicates, []predicate.Predicate{}...)
		if err := blder.ctrl.Watch(src, hdler, allPredicates...); err != nil {
			return err
		}
	}

	// handle Watch
	for gvk := range blder.ceCtx.GetFOW(ccsyntax.FOWWatch) {
		//var obj client.Object
		obj := meta.GetUnstructuredFromGVK(gvk)

		allPredicates := append([]predicate.Predicate(nil), blder.globalPredicates...)
		allPredicates = append(allPredicates, []predicate.Predicate{}...)

		// If the source of this watch is of type *source.Kind, project it.
		src := &source.Kind{Type: obj}

		// TODO replace nil with a real EventHandler where a fn is inserted
		if err := blder.ctrl.Watch(src, nil, allPredicates...); err != nil {
			return err
		}
		/*
			if err := blder.ctrl.Watch(src, w.Eventhandler, allPredicates...); err != nil {
				return err
			}
		*/
	}

	return nil
}

func (blder *builder) getControllerName(gvk schema.GroupVersionKind) string {
	if blder.ceCtx.GetName() != "" {
		return blder.ceCtx.GetName()
	}
	return strings.ToLower(gvk.Kind)
}

func (blder *builder) doController(r reconcile.Reconciler) error {
	globalOpts := blder.mgr.GetControllerOptions()

	ctrlOptions := blder.ctrlOptions
	if ctrlOptions.Reconciler == nil {
		ctrlOptions.Reconciler = r
	}

	gvk := blder.ceCtx.GetForGVK()
	// Setup concurrency.
	/*
	if ctrlOptions.MaxConcurrentReconciles == 0 {
		groupKind := gvk.GroupKind().String()

		if concurrency, ok := globalOpts.GroupKindConcurrency[groupKind]; ok && concurrency > 0 {
			ctrlOptions.MaxConcurrentReconciles = concurrency
		}
	}
	*/

	// Setup cache sync timeout.
	if ctrlOptions.CacheSyncTimeout == 0 && globalOpts.CacheSyncTimeout != nil {
		ctrlOptions.CacheSyncTimeout = *globalOpts.CacheSyncTimeout
	}

	controllerName := blder.getControllerName(gvk)

	// Setup the logger.
	if ctrlOptions.LogConstructor == nil {
		log := blder.mgr.GetLogger().WithValues(
			"controller", controllerName,
			"controllerGroup", gvk.Group,
			"controllerKind", gvk.Kind,
		)

		ctrlOptions.LogConstructor = func(req *reconcile.Request) logr.Logger {
			log := log
			if req != nil {
				log = log.WithValues(
					gvk.Kind, klog.KRef(req.Namespace, req.Name),
					"namespace", req.Namespace, "name", req.Name,
				)
			}
			return log
		}
	}
	// Build the controller and return.
	var err error
	blder.ctrl, err = newController(controllerName, blder.mgr, ctrlOptions)
	return err
}
