/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// this function is assumed to be executed after validation
// validate check if the for is present
func (r *ControllerConfig) GetRootVertexName() string {
	for vertexName := range r.Spec.Properties.For {
		return vertexName
	}
	return ""
}

func (r *ControllerConfig) GetForGvr() ([]schema.GroupVersionResource, error) {
	gvrs, err := r.getGvrList(r.Spec.Properties.For)
	if err != nil {
		return nil, err
	}
	// there should only be 1 for gvr
	return gvrs, nil
}

func (r *ControllerConfig) GetOwnGvrs() ([]schema.GroupVersionResource, error) {
	gvrs, err := r.getGvrList(r.Spec.Properties.Own)
	if err != nil {
		return nil, err
	}
	return gvrs, nil
}

func (r *ControllerConfig) GetWatchGvrs() ([]schema.GroupVersionResource, error) {
	gvrs, err := r.getGvrList(r.Spec.Properties.Watch)
	if err != nil {
		return nil, err
	}
	return gvrs, nil
}

func (r *ControllerConfig) getGvrList(gvrObjs map[string]ControllerConfigGvrObject) ([]schema.GroupVersionResource, error) {
	gvrs := make([]schema.GroupVersionResource, 0, len(gvrObjs))
	for _, gvrObj := range gvrObjs {
		gvr, err := GetGVR(gvrObj.Gvr)
		if err != nil {
			return nil, err
		}
		gvrs = append(gvrs, *gvr)
	}
	return gvrs, nil
}

func GetIdxName(idxName string) (string, int) {
	split := strings.Split(idxName, "/")
	idx, _ := strconv.Atoi(split[1])
	return split[0], idx
}

func GetGVR(gvr *ControllerConfigGvr) (*schema.GroupVersionResource, error) {
	split := strings.Split(gvr.ApiVersion, "/")
	if len(split) != 2 {
		return nil, fmt.Errorf("expecting a GVR apiVersion in format <group>/<version> got: %s", gvr.ApiVersion)
	}
	return &schema.GroupVersionResource{
		Group:    split[0],
		Version:  split[1],
		Resource: gvr.Resource,
	}, nil
}

func (v ControllerConfigPipelineBlock) IsBlock() bool {
	if v.Range == nil && v.Condition == nil {
		return false
	}
	return true
}
