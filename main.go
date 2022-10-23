package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/pkg/profile"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/builder"
	"github.com/yndd/lcnc-runtime/pkg/controllers/reconciler"
	"github.com/yndd/lcnc-runtime/pkg/manager"
	"github.com/yndd/ndd-runtime/pkg/logging"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var debug bool
	var profiler bool
	var concurrency int
	var pollInterval time.Duration
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.IntVar(&concurrency, "concurrency", 1, "Number of items to process simultaneously")
	flag.DurationVar(&pollInterval, "poll-interval", 1*time.Minute, "Poll interval controls how often an individual resource should be checked for drift.")
	flag.BoolVar(&debug, "debug", true, "Enable debug")
	flag.BoolVar(&profiler, "profile", false, "Enable profiler")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	zlog := zap.New(zap.UseDevMode(debug), zap.JSONEncoder())
	ctrl.SetLogger(zlog)
	logger := logging.NewLogrLogger(zlog.WithName("lcnc runtime"))
	if profiler {
		defer profile.Start().Stop()
		go func() {
			http.ListenAndServe(":8000", nil)
		}()
	}

	// Parse config map

	mgr, err := manager.New(ctrl.GetConfigOrDie(), manager.Options{
		Namespace: os.Getenv("POD_NAMESPACE"),
	})
	if err != nil {
		logger.Debug("unable to create manager", "error", err)
		os.Exit(1)
	}

	ctrlcfg := ctrlcfgv1.ControllerConfig{
		For: &ctrlcfgv1.ControllerPipeline{
			Gvr: &ctrlcfgv1.ControllerGroupVersionResource{
				Group:    "admin.yndd.io",
				Version:  "v1alpha1",
				Resource: "tenants",
			},
		},
	}

	gvr := schema.GroupVersionResource{
		Group:    ctrlcfg.For.Gvr.Group,
		Version:  ctrlcfg.For.Gvr.Version,
		Resource: ctrlcfg.For.Gvr.Resource,
	}
	//logger.Debug("gvr", "value", gvr)
	gvk, err := mgr.GetRESTMapper().KindFor(gvr)
	if err != nil {
		logger.Debug("Cannot get gvk", "error", err)
		os.Exit(1)
	}
	//logger.Debug("gvk", "value", gvk)

	b := builder.New(mgr, ctrlcfg)

	_, err = b.Build(reconciler.New(&reconciler.ReconcileInfo{
		Client:       mgr.GetClient(),
		PollInterval: 1 * time.Minute,
		Gvk:          gvk,
		Log:          logger,
	}))
	if err != nil {
		logger.Debug("Cannot build controller", "error", err)
		os.Exit(1)
	}

	//if err := mgr.Add(ctrlr); err != nil {
	//	logger.Debug("Cannot add controller to manager", "error", err)
	//	os.Exit(1)
	//}

	logger.Debug("starting controller manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Debug("problem running manager", "error", err)
		os.Exit(1)
	}
}
