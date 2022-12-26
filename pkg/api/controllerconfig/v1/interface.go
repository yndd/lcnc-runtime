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
	"encoding/json"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

func (r *ControllerConfig) GetForGvk() ([]*schema.GroupVersionKind, error) {
	gvks, err := r.getGvkList(r.Spec.Properties.For)
	if err != nil {
		return nil, err
	}
	// there should only be 1 for gvr
	return gvks, nil
}

func (r *ControllerConfig) GetOwnGvks() ([]*schema.GroupVersionKind, error) {
	gvks, err := r.getGvkList(r.Spec.Properties.Own)
	if err != nil {
		return nil, err
	}
	return gvks, nil
}

func (r *ControllerConfig) GetWatchGvks() ([]*schema.GroupVersionKind, error) {
	gvks, err := r.getGvkList(r.Spec.Properties.Watch)
	if err != nil {
		return nil, err
	}
	return gvks, nil
}

func (r *ControllerConfig) getGvkList(gvrObjs map[string]*GvkObject) ([]*schema.GroupVersionKind, error) {
	gvks := make([]*schema.GroupVersionKind, 0, len(gvrObjs))
	for _, gvrObj := range gvrObjs {
		gvk, err := GetGVK(gvrObj.Resource)
		if err != nil {
			return nil, err
		}
		gvks = append(gvks, gvk)
	}
	return gvks, nil
}

func GetIdxName(idxName string) (string, int) {
	split := strings.Split(idxName, "/")
	idx, _ := strconv.Atoi(split[1])
	return split[0], idx
}

func GetGVK(gvr runtime.RawExtension) (*schema.GroupVersionKind, error) {
	//fmt.Println(string(gvr.Raw))
	var u unstructured.Unstructured
	if err := json.Unmarshal(gvr.Raw, &u); err != nil {
		return nil, err
	}
	gv, err := schema.ParseGroupVersion(u.GetAPIVersion())
	if err != nil {
		return nil, err
	}

	return &schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    u.GetKind(),
	}, nil
}

func (v *Function) HasBlock() bool {
	return v.Block.Range != nil || v.Block.Condition != nil
}

func (v *Block) HasRange() bool {
	if v.Range != nil {
		return true
	}
	if v.Condition == nil {
		return false
	}
	return v.Condition.Block.HasRange()
}

func (r *ControllerConfig) GetPipeline(s string) *Pipeline {
	for _, pipeline := range r.Spec.Properties.Pipelines {
		if pipeline.Name == s {
			return pipeline
		}
	}
	return nil
}
