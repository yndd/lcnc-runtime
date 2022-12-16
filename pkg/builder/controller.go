package builder

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/yndd/lcnc-runtime/pkg/controller"
	"github.com/yndd/lcnc-runtime/pkg/manager"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
)

var newController = controller.New

type Builder interface {
	Build(r reconcile.Reconciler) (controller.Controller, error)
}

/*
type ControllerPipeline struct {
	Gvr          *schema.GroupVersionResource
	Fn           *Function
	Predicates   []predicate.Predicate
	Eventhandler handler.EventHandler
}
*/

type builder struct {
	cfg              ctrlcfgv1.ControllerConfig
	mgr              manager.Manager
	globalPredicates []predicate.Predicate
	ctrl             controller.Controller
	ctrlOptions      controller.Options
}

func New(mgr manager.Manager, cfg ctrlcfgv1.ControllerConfig) Builder {
	b := &builder{
		mgr: mgr,
		cfg: cfg,
	}
	return b
}

func (blder *builder) Build(r reconcile.Reconciler) (controller.Controller, error) {
	if blder.mgr == nil {
		return nil, fmt.Errorf("must provide a non-nil Manager")
	}
	/*
		if blder.cfg.Spec.Properties.For == nil {
			return nil, fmt.Errorf("must provide a non-nil For")
		}
		// Checking the reconcile gvr exist or not
		if blder.cfg.For.Gvr == nil {
			return nil, fmt.Errorf("must provide a gvr for reconciliation")
		}
	*/
	//if blder.c.For.Fn == "" {
	//	return nil, fmt.Errorf("must provide a function for reconciliation")
	//}
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
	// Reconcile type

	gvk, err := blder.cfg.GetForGvk()
	if err != nil {
		return err
	}
	typeForSrc := getUnstructuredObj(gvk[0])

	src := &source.Kind{Type: typeForSrc}
	hdler := &handler.EnqueueRequestForObject{}
	allPredicates := append(blder.globalPredicates, []predicate.Predicate{}...)
	if err := blder.ctrl.Watch(src, hdler, allPredicates...); err != nil {
		return err
	}

	// Watches the managed types
	ownGvks, err := blder.cfg.GetOwnGvks()
	if err != nil {
		return nil
	}
	for _, gvk := range ownGvks {
		obj := getUnstructuredObj(gvk)

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

	// Do the watch requests
	watchGvks, err := blder.cfg.GetWatchGvks()
	if err != nil {
		return nil
	}
	for _, gvk := range watchGvks {
		//var obj client.Object
		obj := getUnstructuredObj(gvk)

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
	if blder.cfg.Name != "" {
		return blder.cfg.Name
	}
	return strings.ToLower(gvk.Kind)
}

func (blder *builder) doController(r reconcile.Reconciler) error {
	globalOpts := blder.mgr.GetControllerOptions()

	ctrlOptions := blder.ctrlOptions
	if ctrlOptions.Reconciler == nil {
		ctrlOptions.Reconciler = r
	}

	gvk, err := blder.cfg.GetForGvk()
	if err != nil {
		return err
	}

	// Setup concurrency.
	if ctrlOptions.MaxConcurrentReconciles == 0 {
		groupKind := gvk[0].GroupKind().String()

		if concurrency, ok := globalOpts.GroupKindConcurrency[groupKind]; ok && concurrency > 0 {
			ctrlOptions.MaxConcurrentReconciles = concurrency
		}
	}

	// Setup cache sync timeout.
	if ctrlOptions.CacheSyncTimeout == 0 && globalOpts.CacheSyncTimeout != nil {
		ctrlOptions.CacheSyncTimeout = *globalOpts.CacheSyncTimeout
	}

	controllerName := blder.getControllerName(gvk[0])

	// Setup the logger.
	if ctrlOptions.LogConstructor == nil {
		log := blder.mgr.GetLogger().WithValues(
			"controller", controllerName,
			"controllerGroup", gvk[0].Group,
			"controllerKind", gvk[0].Kind,
		)

		ctrlOptions.LogConstructor = func(req *reconcile.Request) logr.Logger {
			log := log
			if req != nil {
				log = log.WithValues(
					gvk[0].Kind, klog.KRef(req.Namespace, req.Name),
					"namespace", req.Namespace, "name", req.Name,
				)
			}
			return log
		}
	}

	// Build the controller and return.
	blder.ctrl, err = newController(controllerName, blder.mgr, ctrlOptions)
	return err
}

/*
func getGVKfromGVR(c *rest.Config, gvr schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	mapper, err := apiutil.NewDynamicRESTMapper(c)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	gvk, err := mapper.KindFor(gvr)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return gvk, nil
}
*/

func getUnstructuredObj(gvk schema.GroupVersionKind) *unstructured.Unstructured {
	var u unstructured.Unstructured
	u.SetAPIVersion(gvk.GroupVersion().String())
	u.SetKind(gvk.Kind)
	uCopy := u.DeepCopy()
	return uCopy
}
