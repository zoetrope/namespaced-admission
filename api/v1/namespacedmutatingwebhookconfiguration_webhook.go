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
	apiadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var namespacedmutatingwebhookconfigurationlog = logf.Log.WithName("namespacedmutatingwebhookconfiguration-resource")

func (r *NamespacedMutatingWebhookConfiguration) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-admissionregistration-zoetrope-github-io-v1-namespacedmutatingwebhookconfiguration,mutating=true,failurePolicy=fail,sideEffects=None,groups=admissionregistration.zoetrope.github.io,resources=namespacedmutatingwebhookconfigurations,verbs=create;update,versions=v1,name=mnamespacedmutatingwebhookconfiguration.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &NamespacedMutatingWebhookConfiguration{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NamespacedMutatingWebhookConfiguration) Default() {
	namespacedmutatingwebhookconfigurationlog.Info("default", "name", r.Name)
	for i := range r.Webhooks {
		for j := range r.Webhooks[i].Rules {
			scope := apiadmissionregistrationv1.NamespacedScope
			r.Webhooks[i].Rules[j].Scope = &scope
		}
	}
}
