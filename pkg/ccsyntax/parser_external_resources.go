package ccsyntax

import (
	"sync"

	ctrlcfgv1 "github.com/yndd/lcnc-runtime/pkg/api/controllerconfig/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) GetExternalResources() ([]schema.GroupVersionResource, []Result) {
	er := &er{
		result:    []Result{},
		resources: []schema.GroupVersionResource{},
	}
	er.resultFn = er.recordResult
	er.addResourceFn = er.addResource

	fnc := &WalkConfig{
		gvrObjectFn: er.getGvr,
		functionFn:  er.getFunctionGvr,
	}

	// validate the external resources
	r.walkLcncConfig(fnc)
	return er.resources, er.result
}

type er struct {
	mr            sync.RWMutex
	result        []Result
	resultFn      recordResultFn
	mrs           sync.RWMutex
	resources     []schema.GroupVersionResource
	addResourceFn erAddResourceFn
}

type erAddResourceFn func(schema.GroupVersionResource)

func (r *er) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *er) addResource(er schema.GroupVersionResource) {
	r.mrs.Lock()
	defer r.mrs.Unlock()
	found := false
	for _, resource := range r.resources {
		if resource.Group == er.Group &&
			resource.Version == er.Version &&
			resource.Resource == er.Resource {
			return
		}
	}
	if !found {
		r.resources = append(r.resources, er)
	}
}

func (r *er) addGvr(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvr) {
	gvr, err := ctrlcfgv1.GetGVR(v)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	r.addResource(*gvr)
}

func (r *er) getGvr(oc *OriginContext, v *ctrlcfgv1.ControllerConfigGvrObject) {
	r.addGvr(oc, v.Gvr)
}

func (r *er) getFunctionGvr(oc *OriginContext, v *ctrlcfgv1.ControllerConfigFunction) {
	if v.Input.Gvr != nil {
		r.addGvr(oc, v.Input.Gvr)
	}
	for _, v := range v.Output {
		if v.Gvr != nil {
			r.addGvr(oc, v.Gvr)
		}
	}
}
