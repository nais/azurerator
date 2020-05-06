// +build !ignore_autogenerated

/*


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

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdApplication) DeepCopyInto(out *AzureAdApplication) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdApplication.
func (in *AzureAdApplication) DeepCopy() *AzureAdApplication {
	if in == nil {
		return nil
	}
	out := new(AzureAdApplication)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzureAdApplication) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdApplicationList) DeepCopyInto(out *AzureAdApplicationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AzureAdApplication, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdApplicationList.
func (in *AzureAdApplicationList) DeepCopy() *AzureAdApplicationList {
	if in == nil {
		return nil
	}
	out := new(AzureAdApplicationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzureAdApplicationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdApplicationSpec) DeepCopyInto(out *AzureAdApplicationSpec) {
	*out = *in
	if in.ReplyUrls != nil {
		in, out := &in.ReplyUrls, &out.ReplyUrls
		*out = make([]AzureAdReplyUrl, len(*in))
		copy(*out, *in)
	}
	if in.PreAuthorizedApplications != nil {
		in, out := &in.PreAuthorizedApplications, &out.PreAuthorizedApplications
		*out = make([]AzureAdPreAuthorizedApplication, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdApplicationSpec.
func (in *AzureAdApplicationSpec) DeepCopy() *AzureAdApplicationSpec {
	if in == nil {
		return nil
	}
	out := new(AzureAdApplicationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdApplicationStatus) DeepCopyInto(out *AzureAdApplicationStatus) {
	*out = *in
	in.ProvisionStateTime.DeepCopyInto(&out.ProvisionStateTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdApplicationStatus.
func (in *AzureAdApplicationStatus) DeepCopy() *AzureAdApplicationStatus {
	if in == nil {
		return nil
	}
	out := new(AzureAdApplicationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdPreAuthorizedApplication) DeepCopyInto(out *AzureAdPreAuthorizedApplication) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdPreAuthorizedApplication.
func (in *AzureAdPreAuthorizedApplication) DeepCopy() *AzureAdPreAuthorizedApplication {
	if in == nil {
		return nil
	}
	out := new(AzureAdPreAuthorizedApplication)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdReplyUrl) DeepCopyInto(out *AzureAdReplyUrl) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdReplyUrl.
func (in *AzureAdReplyUrl) DeepCopy() *AzureAdReplyUrl {
	if in == nil {
		return nil
	}
	out := new(AzureAdReplyUrl)
	in.DeepCopyInto(out)
	return out
}
