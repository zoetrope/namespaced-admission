/*
Copyright 2021.

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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NamespacedMutatingWebhookSpec defines the desired state of NamespacedMutatingWebhook
type NamespacedMutatingWebhookSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of NamespacedMutatingWebhook. Edit namespacedmutatingwebhook_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// NamespacedMutatingWebhookStatus defines the observed state of NamespacedMutatingWebhook
type NamespacedMutatingWebhookStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NamespacedMutatingWebhook is the Schema for the namespacedmutatingwebhooks API
type NamespacedMutatingWebhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespacedMutatingWebhookSpec   `json:"spec,omitempty"`
	Status NamespacedMutatingWebhookStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NamespacedMutatingWebhookList contains a list of NamespacedMutatingWebhook
type NamespacedMutatingWebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespacedMutatingWebhook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NamespacedMutatingWebhook{}, &NamespacedMutatingWebhookList{})
}
