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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// Group in the kubernetes api
	Group = "lcnc.yndd.io"
	// Version in the kubernetes api
	Version = "v1"
)

var SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

// Takes an unqualified kind and returns a group-qualified GroupKind
// Note: Generator *requires* it to be called "Kind"
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group-qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

//func init() {
//	SchemeBuilder.Register(addKnownTypes)
//}

// Registers this API group and version to a scheme
// Note: Generator *requires* it to be called "AddToScheme"
var AddToScheme = schemeBuilder.AddToScheme

var schemeBuilder = runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&ControllerConfig{},
		&ControllerConfigList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
})
