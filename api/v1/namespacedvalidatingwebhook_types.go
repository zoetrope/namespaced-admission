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
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	admissionv1apply "k8s.io/client-go/applyconfigurations/admissionregistration/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ValidatingWebhook describes an admission webhook and the resources and operations it applies to.
type ValidatingWebhook struct {
	// Name is the name of the admission webhook.
	//+kubebuilder:validation:Required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// ClientConfig defines how to communicate with the hook.
	//+kubebuilder:validation:Required
	ClientConfig admissionv1apply.WebhookClientConfigApplyConfiguration `json:"clientConfig"`

	// Rules describes what operations on what resources/subresources the webhook cares about.
	Rules []*admissionv1apply.RuleWithOperationsApplyConfiguration `json:"rules,omitempty"`

	// FailurePolicy defines how unrecognized errors from the admission endpoint are handled -
	// allowed values are Ignore or Fail. Defaults to Fail.
	// +kubebuilder:default="Fail"
	// +optional
	FailurePolicy *admissionv1.FailurePolicyType `json:"failurePolicy,omitempty"`

	// matchPolicy defines how the "rules" list is used to match incoming requests.
	// Allowed values are "Exact" or "Equivalent".
	// +kubebuilder:default="Equivalent"
	// +optional
	MatchPolicy *admissionv1.MatchPolicyType `json:"matchPolicy,omitempty"`

	// ObjectSelector decides whether to run the webhook based on if the
	// object has matching labels.
	// +optional
	ObjectSelector *metav1apply.LabelSelectorApplyConfiguration `json:"objectSelector,omitempty"`

	// SideEffects states whether this webhook has side effects.
	SideEffects *admissionv1.SideEffectClass `json:"sideEffects"`

	// TimeoutSeconds specifies the timeout for this webhook. After the timeout passes,
	// +kubebuilder:default=10
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`

	// AdmissionReviewVersions is an ordered list of preferred `AdmissionReview`
	// versions the Webhook expects.
	AdmissionReviewVersions []string `json:"admissionReviewVersions"`
}

// NamespacedValidatingWebhookStatus defines the observed state of NamespacedValidatingWebhook
//+kubebuilder:validation:Enum=Applied
type NamespacedValidatingWebhookStatus string

const (
	NamespacedValidatingWebhookApplied = NamespacedValidatingWebhookStatus("Applied")
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NamespacedValidatingWebhook is the Schema for the namespacedvalidatingwebhooks API
type NamespacedValidatingWebhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Webhooks is a list of webhooks and the affected resources and operations.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Webhooks []ValidatingWebhook `json:"webhooks,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	Status NamespacedValidatingWebhookStatus `json:"status,omitempty"`
}

func (w NamespacedValidatingWebhook) ConfigName() string {
	return w.Namespace + "-" + w.Name
}

//+kubebuilder:object:root=true

// NamespacedValidatingWebhookList contains a list of NamespacedValidatingWebhook
type NamespacedValidatingWebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespacedValidatingWebhook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NamespacedValidatingWebhook{}, &NamespacedValidatingWebhookList{})
}
