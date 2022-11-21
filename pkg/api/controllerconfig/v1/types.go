package v1

import "sigs.k8s.io/controller-runtime/pkg/handler"

type ControllerConfig struct {
	Name string `yaml:"image,omitempty" json:"image,omitempty"`
	//Mgr              manager.Manager
	//GlobalPredicates []predicate.Predicate
	For   *ControllerPipeline   `yaml:"for" json:"for"`
	Owns  []*ControllerPipeline `yaml:"owns,omitempty" json:"owns,omitempty"`
	Watch []*ControllerPipeline `yaml:"watch,omitempty" json:"watch,omitempty"`
	//Ctrl             controller.Controller
	//CtrlOptions      controller.Options
}

type ControllerPipeline struct {
	Gvr *ControllerGroupVersionResource `yaml:"gvr" json:"gvr"`
	Fn  []*Function                       `yaml:"function" json:"function"`
	//Predicates   []predicate.Predicate
	Eventhandler handler.EventHandler
}

type ControllerGroupVersionResource struct {
	Group    string `yaml:"group,omitempty" json:"group,omitempty"`
	Version  string `yaml:"version,omitempty" json:"version,omitempty"`
	Resource string `yaml:"resurce,omitempty" json:"resource,omitempty"`
}

type Function struct {
	// `Image` specifies the function container image.
	// It can either be fully qualified, e.g.:
	//
	//    image: docker.io/set-topology
	//
	// Optionally, kpt can be configured to use a image
	// registry host-path that will be used to resolve the image path in case
	// the image path is missing (Defaults to docker.io/yndd).
	// e.g. The following resolves to docker.io/yndd/set-topology:
	//
	//    image: set-topology
	Image string `yaml:"image,omitempty" json:"image,omitempty"`

	// Exec specifies the function binary executable.
	// The executable can be fully qualified or it must exists in the $PATH e.g:
	//
	//      exec: set-topology
	//      exec: /usr/local/bin/my-custom-fn
	Exec string `yaml:"exec,omitempty" json:"exec,omitempty"`
}
