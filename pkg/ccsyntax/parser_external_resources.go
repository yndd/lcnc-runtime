package ccsyntax

import (
	"fmt"
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) GetExternalResources() ([]schema.GroupVersionKind, []Result) {
	er := &er{
		result:    []Result{},
		resources: []schema.GroupVersionKind{},
	}
	er.resultFn = er.recordResult
	er.addKindFn = er.addKind

	fnc := &WalkConfig{
		gvkObjectFn: er.getGvk,
		functionFn:  er.getFunctionGvk,
	}

	// validate the external resources
	r.walkLcncConfig(fnc)
	return er.resources, er.result
}

type er struct {
	mr        sync.RWMutex
	result    []Result
	resultFn  recordResultFn
	mrs       sync.RWMutex
	resources []schema.GroupVersionKind
	addKindFn erAddKindFn
}

type erAddKindFn func(schema.GroupVersionKind)

func (r *er) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *er) addKind(er schema.GroupVersionKind) {
	fmt.Printf("add kind: %v \n", er)
	r.mrs.Lock()
	defer r.mrs.Unlock()
	found := false
	for _, resource := range r.resources {
		if resource.Group == er.Group &&
			resource.Version == er.Version &&
			resource.Kind == er.Kind {
			return
		}
	}
	if !found {
		r.resources = append(r.resources, er)
	}
}

func (r *er) addGvk(oc *OriginContext, v runtime.RawExtension) {
	gvk, err := ctrlcfgv1.GetGVK(v)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
		return
	}
	r.addKind(gvk)
}

func (r *er) getGvk(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvkObject) {
	r.addGvk(oc, v.Resource)
}

func (r *er) getFunctionGvk(oc *OriginContext, v *ctrlcfgv1.ControllerConfigFunction) {
	if len(v.Input.Resource.Raw) != 0 {
		r.addGvk(oc, v.Input.Resource)
	}
	for _, v := range v.Output {
		if len(v.Resource.Raw) != 0 {
			r.addGvk(oc, v.Resource)
		}
	}
}
