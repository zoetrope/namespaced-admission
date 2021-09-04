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

package controllers

import (
	"context"

	webhookv1 "github.com/zoetrope/namespaced-webhook/api/v1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	admissionv1apply "k8s.io/client-go/applyconfigurations/admissionregistration/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NamespacedMutatingWebhookReconciler reconciles a NamespacedMutatingWebhook object
type NamespacedMutatingWebhookReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webhook.zoetrope.github.io,resources=namespacedmutatingwebhooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webhook.zoetrope.github.io,resources=namespacedmutatingwebhooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webhook.zoetrope.github.io,resources=namespacedmutatingwebhooks/finalizers,verbs=update
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfiguration,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespacedMutatingWebhook object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *NamespacedMutatingWebhookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var nmw webhookv1.NamespacedMutatingWebhook
	err := r.Get(ctx, req.NamespacedName, &nmw)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.reconcileWebhookConfiguration(ctx, nmw)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NamespacedMutatingWebhookReconciler) reconcileWebhookConfiguration(ctx context.Context, nmw webhookv1.NamespacedMutatingWebhook) error {
	logger := log.FromContext(ctx)

	controllerName := "namespaced-mutating-webhook-controller"
	configName := nmw.Namespace + "-" + nmw.Name

	config := admissionv1apply.MutatingWebhookConfiguration(configName)
	webhooks := make([]*admissionv1apply.MutatingWebhookApplyConfiguration, 0)
	for _, hook := range nmw.Webhooks {
		webhook := admissionv1apply.MutatingWebhook().
			WithName(hook.Name).
			WithClientConfig(&hook.ClientConfig).
			WithRules(hook.Rules...).
			WithObjectSelector(hook.ObjectSelector).
			WithAdmissionReviewVersions(hook.AdmissionReviewVersions...)
		if hook.FailurePolicy != nil {
			webhook.WithFailurePolicy(*hook.FailurePolicy)
		}
		if hook.MatchPolicy != nil {
			webhook.WithMatchPolicy(*hook.MatchPolicy)
		}
		if hook.SideEffects != nil {
			webhook.WithSideEffects(*hook.SideEffects)
		}
		if hook.TimeoutSeconds != nil {
			webhook.WithTimeoutSeconds(*hook.TimeoutSeconds)
		}
		if hook.ReinvocationPolicy != nil {
			webhook.WithReinvocationPolicy(*hook.ReinvocationPolicy)
		}
		webhook.WithNamespaceSelector(metav1apply.LabelSelector().
			WithMatchExpressions(metav1apply.LabelSelectorRequirement().
				WithKey("kubernetes.io/metadata.name").
				WithOperator(metav1.LabelSelectorOpIn).
				WithValues(nmw.Namespace),
			),
		)
		webhooks = append(webhooks, webhook)
	}
	config.WithWebhooks(webhooks...)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(config)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current admissionv1.MutatingWebhookConfiguration
	err = r.Get(ctx, client.ObjectKey{Name: configName}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := admissionv1apply.ExtractMutatingWebhookConfiguration(&current, controllerName)
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(config, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: controllerName,
		Force:        pointer.Bool(true),
	})
	if err != nil {
		logger.Error(err, "unable to create or update MutatingWebhookConfiguration")
		return err
	}

	logger.Info("reconcile MutatingWebhookConfiguration successfully")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespacedMutatingWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webhookv1.NamespacedMutatingWebhook{}).
		Owns(&admissionv1.MutatingWebhookConfiguration{}). //TODO: FIXME
		Complete(r)
}
