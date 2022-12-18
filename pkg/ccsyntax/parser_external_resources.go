package ccsyntax

import (
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
	er.addKindFn = er.addGVK

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

func (r *er) addGVK(gvk schema.GroupVersionKind) {
	//fmt.Printf("add gvk: %v \n", gvk)
	r.mrs.Lock()
	defer r.mrs.Unlock()
	found := false
	for _, resource := range r.resources {
		if resource.Group == gvk.Group &&
			resource.Version == gvk.Version &&
			resource.Kind == gvk.Kind {
			return
		}
	}
	if !found {
		r.resources = append(r.resources, gvk)
	}
}

func (r *er) addGvk(gvk schema.GroupVersionKind) {
	r.addGVK(gvk)
}

func (r *er) getGvk(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvkObject) schema.GroupVersionKind {
	gvk := r.getgvk(oc, v.Resource)
	r.addGvk(gvk)
	return gvk
}

func (r *er) getFunctionGvk(oc *OriginContext, v *ctrlcfgv1.ControllerConfigFunction) {
	if len(v.Input.Resource.Raw) != 0 {
		gvk := r.getgvk(oc, v.Input.Resource)
		r.addGvk(gvk)
	}
	for _, v := range v.Output {
		if len(v.Resource.Raw) != 0 {
			gvk := r.getgvk(oc, v.Resource)
			r.addGvk(gvk)
		}
	}
}

func (r *er) getgvk(oc *OriginContext, v runtime.RawExtension) schema.GroupVersionKind {
	gvk, err := ctrlcfgv1.GetGVK(v)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
		return gvk
	}
	return gvk
}
