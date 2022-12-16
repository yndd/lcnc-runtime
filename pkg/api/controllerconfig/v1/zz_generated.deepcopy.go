//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2021 NDD.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Block) DeepCopyInto(out *Block) {
	*out = *in
	if in.Range != nil {
		in, out := &in.Range, &out.Range
		*out = new(RangeValue)
		(*in).DeepCopyInto(*out)
	}
	if in.Condition != nil {
		in, out := &in.Condition, &out.Condition
		*out = new(ConditionExpression)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Block.
func (in *Block) DeepCopy() *Block {
	if in == nil {
		return nil
	}
	out := new(Block)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConditionExpression) DeepCopyInto(out *ConditionExpression) {
	*out = *in
	in.Block.DeepCopyInto(&out.Block)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConditionExpression.
func (in *ConditionExpression) DeepCopy() *ConditionExpression {
	if in == nil {
		return nil
	}
	out := new(ConditionExpression)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfig) DeepCopyInto(out *ControllerConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfig.
func (in *ControllerConfig) DeepCopy() *ControllerConfig {
	if in == nil {
		return nil
	}
	out := new(ControllerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ControllerConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfigFunction) DeepCopyInto(out *ControllerConfigFunction) {
	*out = *in
	in.Block.DeepCopyInto(&out.Block)
	in.Executor.DeepCopyInto(&out.Executor)
	if in.Vars != nil {
		in, out := &in.Vars, &out.Vars
		*out = make(map[string]*ControllerConfigFunction, len(*in))
		for key, val := range *in {
			var outVal *ControllerConfigFunction
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(ControllerConfigFunction)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	if in.Input != nil {
		in, out := &in.Input, &out.Input
		*out = new(ControllerConfigInput)
		(*in).DeepCopyInto(*out)
	}
	if in.Output != nil {
		in, out := &in.Output, &out.Output
		*out = make(map[string]*ControllerConfigOutput, len(*in))
		for key, val := range *in {
			var outVal *ControllerConfigOutput
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(ControllerConfigOutput)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfigFunction.
func (in *ControllerConfigFunction) DeepCopy() *ControllerConfigFunction {
	if in == nil {
		return nil
	}
	out := new(ControllerConfigFunction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfigGvkObject) DeepCopyInto(out *ControllerConfigGvkObject) {
	*out = *in
	in.Resource.DeepCopyInto(&out.Resource)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfigGvkObject.
func (in *ControllerConfigGvkObject) DeepCopy() *ControllerConfigGvkObject {
	if in == nil {
		return nil
	}
	out := new(ControllerConfigGvkObject)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfigInput) DeepCopyInto(out *ControllerConfigInput) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.GenericInput != nil {
		in, out := &in.GenericInput, &out.GenericInput
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.Resource.DeepCopyInto(&out.Resource)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfigInput.
func (in *ControllerConfigInput) DeepCopy() *ControllerConfigInput {
	if in == nil {
		return nil
	}
	out := new(ControllerConfigInput)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfigList) DeepCopyInto(out *ControllerConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ControllerConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfigList.
func (in *ControllerConfigList) DeepCopy() *ControllerConfigList {
	if in == nil {
		return nil
	}
	out := new(ControllerConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ControllerConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfigOutput) DeepCopyInto(out *ControllerConfigOutput) {
	*out = *in
	in.Resource.DeepCopyInto(&out.Resource)
	if in.GenericOutput != nil {
		in, out := &in.GenericOutput, &out.GenericOutput
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfigOutput.
func (in *ControllerConfigOutput) DeepCopy() *ControllerConfigOutput {
	if in == nil {
		return nil
	}
	out := new(ControllerConfigOutput)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfigPipeline) DeepCopyInto(out *ControllerConfigPipeline) {
	*out = *in
	if in.Vars != nil {
		in, out := &in.Vars, &out.Vars
		*out = make(map[string]*ControllerConfigFunction, len(*in))
		for key, val := range *in {
			var outVal *ControllerConfigFunction
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(ControllerConfigFunction)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	if in.Tasks != nil {
		in, out := &in.Tasks, &out.Tasks
		*out = make(map[string]*ControllerConfigFunction, len(*in))
		for key, val := range *in {
			var outVal *ControllerConfigFunction
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(ControllerConfigFunction)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfigPipeline.
func (in *ControllerConfigPipeline) DeepCopy() *ControllerConfigPipeline {
	if in == nil {
		return nil
	}
	out := new(ControllerConfigPipeline)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfigProperties) DeepCopyInto(out *ControllerConfigProperties) {
	*out = *in
	if in.For != nil {
		in, out := &in.For, &out.For
		*out = make(map[string]*ControllerConfigGvkObject, len(*in))
		for key, val := range *in {
			var outVal *ControllerConfigGvkObject
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(ControllerConfigGvkObject)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	if in.Own != nil {
		in, out := &in.Own, &out.Own
		*out = make(map[string]*ControllerConfigGvkObject, len(*in))
		for key, val := range *in {
			var outVal *ControllerConfigGvkObject
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(ControllerConfigGvkObject)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	if in.Watch != nil {
		in, out := &in.Watch, &out.Watch
		*out = make(map[string]*ControllerConfigGvkObject, len(*in))
		for key, val := range *in {
			var outVal *ControllerConfigGvkObject
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(ControllerConfigGvkObject)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	if in.ControllerConfigPipelines != nil {
		in, out := &in.ControllerConfigPipelines, &out.ControllerConfigPipelines
		*out = make([]*ControllerConfigPipeline, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ControllerConfigPipeline)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfigProperties.
func (in *ControllerConfigProperties) DeepCopy() *ControllerConfigProperties {
	if in == nil {
		return nil
	}
	out := new(ControllerConfigProperties)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfigSpec) DeepCopyInto(out *ControllerConfigSpec) {
	*out = *in
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = new(ControllerConfigProperties)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfigSpec.
func (in *ControllerConfigSpec) DeepCopy() *ControllerConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ControllerConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfigStatus) DeepCopyInto(out *ControllerConfigStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfigStatus.
func (in *ControllerConfigStatus) DeepCopy() *ControllerConfigStatus {
	if in == nil {
		return nil
	}
	out := new(ControllerConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Executor) DeepCopyInto(out *Executor) {
	*out = *in
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(string)
		**out = **in
	}
	if in.Exec != nil {
		in, out := &in.Exec, &out.Exec
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Executor.
func (in *Executor) DeepCopy() *Executor {
	if in == nil {
		return nil
	}
	out := new(Executor)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RangeValue) DeepCopyInto(out *RangeValue) {
	*out = *in
	in.Block.DeepCopyInto(&out.Block)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RangeValue.
func (in *RangeValue) DeepCopy() *RangeValue {
	if in == nil {
		return nil
	}
	out := new(RangeValue)
	in.DeepCopyInto(out)
	return out
}
