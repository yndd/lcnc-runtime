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

// CopyVariables copies the variable block and splits multiple entries
// in a slice to a single entry. This allows to build a generic
// resolution processor
func CopyVariables(vars []ControllerConfigVarBlock) map[string]ControllerConfigVarBlock {
	newvars := map[string]ControllerConfigVarBlock{}
	for idx, varBlock := range vars {
		for k, v := range varBlock.ControllerConfigVariables {
			newvars[strings.Join([]string{k, strconv.Itoa(idx)}, "/")] = ControllerConfigVarBlock{
				ControllerConfigBlock: varBlock.ControllerConfigBlock,
				ControllerConfigVariables: map[string]ControllerConfigVar{
					k: v,
				},
			}
		}
	}
	return newvars
}

// CopyFunctions copies the variable block and splits multiple entries
// in a slice to a single entry. This allows to build a generic
// resolution processor
func CopyFunctions(fns []ControllerConfigFunctionsBlock) map[string]ControllerConfigFunctionsBlock {
	newfns := map[string]ControllerConfigFunctionsBlock{}
	for idx, fnBlock := range fns {
		for k, v := range fnBlock.ControllerConfigFunctions {
			newfns[strings.Join([]string{k, strconv.Itoa(idx)}, "/")] = ControllerConfigFunctionsBlock{
				ControllerConfigBlock: fnBlock.ControllerConfigBlock,
				ControllerConfigFunctions: map[string]ControllerConfigFunction{
					k: v,
				},
			}
		}
	}
	return newfns
}

func GetIdxName(idxName string) (string, int) {
	split := strings.Split(idxName, "/")
	idx, _ := strconv.Atoi(split[1])
	return split[0], idx
}

func GetGVR(s string) (*schema.GroupVersionResource, error) {
	split := strings.Split(s, "/")
	if len(split) != 3 {
		return nil, fmt.Errorf("expecting a GVR in format <group>/<version>/<resource>, got: %s", s)
	}
	return &schema.GroupVersionResource{
		Group:    split[0],
		Version:  split[1],
		Resource: split[2],
	}, nil
}
