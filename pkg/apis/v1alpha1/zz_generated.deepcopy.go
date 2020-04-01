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
func (in *AzureAdCredential) DeepCopyInto(out *AzureAdCredential) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdCredential.
func (in *AzureAdCredential) DeepCopy() *AzureAdCredential {
	if in == nil {
		return nil
	}
	out := new(AzureAdCredential)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzureAdCredential) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdCredentialCondition) DeepCopyInto(out *AzureAdCredentialCondition) {
	*out = *in
	if in.Reason != nil {
		in, out := &in.Reason, &out.Reason
		*out = new(string)
		**out = **in
	}
	if in.Message != nil {
		in, out := &in.Message, &out.Message
		*out = new(string)
		**out = **in
	}
	if in.LastHeartbeatTime != nil {
		in, out := &in.LastHeartbeatTime, &out.LastHeartbeatTime
		*out = (*in).DeepCopy()
	}
	if in.LastTransitionTime != nil {
		in, out := &in.LastTransitionTime, &out.LastTransitionTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdCredentialCondition.
func (in *AzureAdCredentialCondition) DeepCopy() *AzureAdCredentialCondition {
	if in == nil {
		return nil
	}
	out := new(AzureAdCredentialCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdCredentialList) DeepCopyInto(out *AzureAdCredentialList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AzureAdCredential, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdCredentialList.
func (in *AzureAdCredentialList) DeepCopy() *AzureAdCredentialList {
	if in == nil {
		return nil
	}
	out := new(AzureAdCredentialList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzureAdCredentialList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdCredentialSpec) DeepCopyInto(out *AzureAdCredentialSpec) {
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdCredentialSpec.
func (in *AzureAdCredentialSpec) DeepCopy() *AzureAdCredentialSpec {
	if in == nil {
		return nil
	}
	out := new(AzureAdCredentialSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureAdCredentialStatus) DeepCopyInto(out *AzureAdCredentialStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]AzureAdCredentialCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PasswordKeyId != nil {
		in, out := &in.PasswordKeyId, &out.PasswordKeyId
		*out = new(string)
		**out = **in
	}
	if in.CertificateKeyId != nil {
		in, out := &in.CertificateKeyId, &out.CertificateKeyId
		*out = new(string)
		**out = **in
	}
	if in.SynchronizationHash != nil {
		in, out := &in.SynchronizationHash, &out.SynchronizationHash
		*out = new(string)
		**out = **in
	}
	if in.SynchronizationTime != nil {
		in, out := &in.SynchronizationTime, &out.SynchronizationTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureAdCredentialStatus.
func (in *AzureAdCredentialStatus) DeepCopy() *AzureAdCredentialStatus {
	if in == nil {
		return nil
	}
	out := new(AzureAdCredentialStatus)
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
