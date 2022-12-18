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
	"k8s.io/apimachinery/pkg/runtime/schema"

	//"github.com/yndd/lcnc-runtime/pkg/builder"
	"github.com/yndd/lcnc-runtime/pkg/ccsyntax"
	//"github.com/yndd/lcnc-runtime/pkg/controllers/reconciler"
	"github.com/yndd/lcnc-runtime/pkg/manager"
	"github.com/yndd/ndd-runtime/pkg/logging"

	//"gopkg.in/yaml.v3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

const yamlFile = "./examples/topo2.yaml"

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

	fb, err := os.ReadFile(yamlFile)
	if err != nil {
		logger.Debug("cannot read file", "error", err)
		os.Exit(1)
	}
	logger.Debug("read file")

	ctrlcfg := &ctrlcfgv1.ControllerConfig{}
	if err := yaml.Unmarshal(fb, ctrlcfg); err != nil {
		logger.Debug("cannot unmarshal", "error", err)
		os.Exit(1)
	}
	logger.Debug("unmarshal succeeded", "cfg", ctrlcfg)

	p, result := ccsyntax.NewParser(ctrlcfg)
	if len(result) > 0 {
		logger.Debug("config syntax validation failed", "result", result)
		os.Exit(1)
	}
	logger.Debug("new parser succeeded")

	ceCtx, result := p.Parse()
	if len(result) != 0 {
		logger.Debug("cannot parse resources", "result", result)
		os.Exit(1)
	}
	logger.Debug("parsing succeeded")


	gvks, result := p.GetExternalResources()
	if len(result) > 0 {
		logger.Debug("config get external resources failed", "result", result)
		os.Exit(1)
	}

	// validate if we can resolve the gvr to gvk in the system
	for _, gvk := range gvks {
		gvk, err := mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
		if err != nil {
			logger.Debug("Cannot get gvk in system", "error", err)
			os.Exit(1)
		}
		logger.Debug("gvk", "value", gvk)
	}

	//s := scheduler.New()
	//s.Walk(context.TODO(), d)
	//s.GetWalkResult()

	b := builder.New(mgr, ceCtx)
	_, err = b.Build(reconciler.New(&reconciler.ReconcileInfo{
		Client:       mgr.GetClient(),
		PollInterval: 1 * time.Minute,
		CeCtx:        ceCtx,
		Log:          logger,
	}))
	if err != nil {
		logger.Debug("Cannot build controller", "error", err)
		os.Exit(1)
	}

	logger.Debug("starting controller manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Debug("problem running manager", "error", err)
		os.Exit(1)
	}
}
