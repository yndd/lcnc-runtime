package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	//"github.com/henderiw-k8s-lcnc/discovery/discovery"
	//"github.com/henderiw-k8s-lcnc/discovery/registrator"
	"github.com/pkg/profile"
	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"github.com/yndd/lcnc-runtime/pkg/builder"
	"github.com/yndd/lcnc-runtime/pkg/controller"
	"github.com/yndd/lcnc-runtime/pkg/controllers/reconciler"
	"github.com/yndd/lcnc-runtime/pkg/exec/fnruntime"
	"go.uber.org/zap/zapcore"

	//"github.com/yndd/lcnc-runtime/pkg/pcache"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/yndd/lcnc-runtime/pkg/ccsyntax"
	"github.com/yndd/lcnc-runtime/pkg/manager"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

//const yamlFile = "./examples/upf.yaml"
const yamlFile = "./examples/topo4.yaml"

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
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	l := ctrl.Log.WithName("lcnc runtime")

	if profiler {
		defer profile.Start().Stop()
		go func() {
			http.ListenAndServe(":8000", nil)
		}()
	}

	mgr, err := manager.New(ctrl.GetConfigOrDie(), manager.Options{
		Namespace: os.Getenv("POD_NAMESPACE"),
	})
	if err != nil {
		l.Error(err, "unable to create manager")
		os.Exit(1)
	}

	fb, err := os.ReadFile(yamlFile)
	if err != nil {
		l.Error(err, "cannot read file")
		os.Exit(1)
	}
	l.Info("read file")

	ctrlcfg := &ctrlcfgv1.ControllerConfig{}
	if err := yaml.Unmarshal(fb, ctrlcfg); err != nil {
		l.Error(err, "cannot unmarshal")
		os.Exit(1)
	}
	l.Info("unmarshal succeeded")

	p, result := ccsyntax.NewParser(ctrlcfg)
	if len(result) > 0 {
		l.Error(err, "ccsyntax validation failed", "result", result)
		os.Exit(1)
	}
	l.Info("ccsyntax validation succeeded")

	ceCtx, result := p.Parse()
	if len(result) != 0 {
		for _, res := range result {
			l.Error(err, "ccsyntax parsing failed", "result", res)
		}
		os.Exit(1)
	}
	l.Info("ccsyntax parsing succeeded")

	gvks, result := p.GetExternalResources()
	if len(result) > 0 {
		l.Error(err, "ccsyntax get external resources failed", "result", result)
		os.Exit(1)
	}

	// validate if we can resolve the gvr to gvk in the system
	for _, gvk := range gvks {
		gvk, err := mgr.GetRESTMapper().RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
		if err != nil {
			l.Error(err, "ccsyntax get gvk mapping in api server", "result", result)
			os.Exit(1)
		}
		l.Info("gvk", "value", gvk)
	}

	ge := make(chan event.GenericEvent)

	b := builder.New(&builder.Config{
		Mgr:          mgr,
		CeCtx:        ceCtx,
		GenericEvent: ge,
	}, controller.Options{
		MaxConcurrentReconciles: 8,
	})
	_, err = b.Build(reconciler.New(&reconciler.Config{
		Client:       mgr.GetClient(),
		PollInterval: 1 * time.Minute,
		CeCtx:        ceCtx,
	}))
	if err != nil {
		l.Error(err, "cannot build controller")
		os.Exit(1)
	}

	l.Info("setup controller")
	ctx := ctrl.SetupSignalHandler()
	/*
		reg, err := registrator.New(ctx, ctrl.GetConfigOrDie(), &registrator.Options{
			ServiceDiscovery:          discovery.ServiceDiscoveryTypeK8s,
			ServiceDiscoveryNamespace: "ipam",
		})
		if err != nil {
			l.Error(err, "Cannot create registrator")
			os.Exit(1)
		}

		// create proxy cache
		c := pcache.New(&pcache.Config{
			Registrator: reg,
			EventChannels: map[schema.GroupVersionKind]chan event.GenericEvent{
				*ceCtx.GetForGVK(): ge,
			},
		})

		c.Start(ctx)
	*/
	//cancelFns := []context.CancelFunc{}
	ctx, cancel := context.WithCancel(ctx)
	for gvk, svcCtx := range ceCtx.GetServices().Get() {
		l.Info("run service", "gvk", gvk)
		runner, err := fnruntime.NewRunner(ctx, svcCtx.Fn,
			fnruntime.RunnerOptions{
				Kind:           fnruntime.FunctionKindService,
				ServicePort:    svcCtx.Port,
				ResolveToImage: fnruntime.ResolveToImageForCLI,
			},
		)
		if err != nil {
			l.Error(err, "cannot create service runner", "gvk", gvk)
			cancel()
			return
		}
		
		g := gvk
		go func() {
			_, err = runner.Run(ctx, nil)
			if err != nil {
				l.Error(err, "cannot run service fn", "gvk", g)
				cancel()
				return
			}
			l.Info("service run stopped", "gvk", g)
			cancel()
		}()
	}

	time.Sleep(2 * time.Second)

	l.Info("starting controller manager")
	if err := mgr.Start(ctx); err != nil {
		l.Error(err, "cannot run manager")
		cancel()
	}
}
