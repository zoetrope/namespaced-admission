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

package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	v1 "github.com/zoetrope/namespaced-admission/api/v1"
	admissionv1 "k8s.io/api/admission/v1"
	apiadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var mutatingLogger = logf.Log.WithName("namespacedmutatingwebhookconfiguration-resource")

//+kubebuilder:webhook:path=/mutate-admissionregistration-zoetrope-github-io-v1-namespacedmutatingwebhookconfiguration,mutating=true,failurePolicy=fail,sideEffects=None,groups=admissionregistration.zoetrope.github.io,resources=namespacedmutatingwebhookconfigurations,verbs=create;update,versions=v1,name=mnamespacedmutatingwebhookconfiguration.kb.io,admissionReviewVersions={v1,v1beta1}

type namespacedMutatingWebhookConfigurationMutator struct {
	dec *admission.Decoder
}

var _ admission.Handler = &namespacedMutatingWebhookConfigurationMutator{}

func (m *namespacedMutatingWebhookConfigurationMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation != admissionv1.Create {
		return admission.Allowed("")
	}

	nmw := &v1.NamespacedMutatingWebhookConfiguration{}
	if err := m.dec.Decode(req, nmw); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	for i := range nmw.Webhooks {
		for j := range nmw.Webhooks[i].Rules {
			scope := apiadmissionregistrationv1.NamespacedScope
			nmw.Webhooks[i].Rules[j].Scope = &scope
		}
	}
	data, err := json.Marshal(nmw)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, data)
}

//+kubebuilder:webhook:path=/validate-admissionregistration-zoetrope-github-io-v1-namespacedmutatingwebhookconfiguration,mutating=false,failurePolicy=fail,sideEffects=None,groups=admissionregistration.zoetrope.github.io,resources=namespacedmutatingwebhookconfigurations,verbs=create;update,versions=v1,name=vnamespacedmutatingwebhookconfiguration.kb.io,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:rbac:groups=core,resources=serviceaccounts;users;groups,verbs=impersonate

type namespacedMutatingWebhookConfigurationValidator struct {
	config *rest.Config
	dec    *admission.Decoder
}

var _ admission.Handler = &namespacedMutatingWebhookConfigurationValidator{}

func (v *namespacedMutatingWebhookConfigurationValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation != admissionv1.Create && req.Operation != admissionv1.Update {
		return admission.Allowed("")
	}

	nmw := &v1.NamespacedMutatingWebhookConfiguration{}
	if err := v.dec.Decode(req, nmw); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	err := v.validate(ctx, nmw, []string{"get", strings.ToLower(string(req.Operation))})

	if err != nil {
		return admission.Denied(err.Error())
	}

	return admission.Allowed("ok")
}

func (v *namespacedMutatingWebhookConfigurationValidator) validate(ctx context.Context, nmw *v1.NamespacedMutatingWebhookConfiguration, verbs []string) error {
	userName := "system:serviceaccount:" + nmw.Namespace + ":" + nmw.ServiceAccountName
	config := rest.CopyConfig(v.config)
	config.Impersonate.UserName = userName
	mutatingLogger.Info("validate", "userName", userName)
	cl, err := kubernetes.NewForConfig(config)
	if err != nil {
		return apierrors.NewInternalError(err)
	}
	authClient := cl.AuthorizationV1()

	var errs field.ErrorList
	for i, hook := range nmw.Webhooks {
		for j, rule := range hook.Rules {
			p := field.NewPath("webhook").Index(i).Child("rules").Index(j)
			for _, group := range rule.APIGroups {
				for _, ver := range rule.APIVersions {
					for _, res := range rule.Resources {
						for _, verb := range verbs {
							accessible, err := canAccess(ctx, authClient, verb, group, ver, res, nmw.Namespace)
							if err != nil {
								return apierrors.NewInternalError(err)
							}
							if !accessible {
								errs = append(errs, field.Forbidden(p, fmt.Sprintf("%s cannot %s %s/%s/%s in %s", userName, verb, group, ver, res, nmw.Namespace)))
							}
						}
					}
				}
			}
		}
	}

	if len(errs) > 0 {
		err := apierrors.NewInvalid(schema.GroupKind{Group: v1.GroupVersion.Group, Kind: "NamespacedMutatingWebhookConfiguration"}, nmw.Name, errs)
		mutatingLogger.Error(err, "validation error", "name", nmw.Name)
		return err
	}

	return nil
}

// SetupNamespacedMutatingWebhookConfigurationWebhook registers the webhooks for NamespacedMutatingWebhookConfiguration
func SetupNamespacedMutatingWebhookConfigurationWebhook(mgr manager.Manager, config *rest.Config, dec *admission.Decoder) {
	serv := mgr.GetWebhookServer()

	m := &namespacedMutatingWebhookConfigurationMutator{
		dec: dec,
	}
	serv.Register("/mutate-admissionregistration-zoetrope-github-io-v1-namespacedmutatingwebhookconfiguration", &webhook.Admission{Handler: m})

	v := &namespacedMutatingWebhookConfigurationValidator{
		config: config,
		dec:    dec,
	}
	serv.Register("/validate-admissionregistration-zoetrope-github-io-v1-namespacedmutatingwebhookconfiguration", &webhook.Admission{Handler: v})
}
