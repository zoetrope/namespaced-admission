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
	"fmt"

	webhookv1 "github.com/zoetrope/namespaced-webhook/api/v1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	admissionv1apply "k8s.io/client-go/applyconfigurations/admissionregistration/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// NamespacedValidatingWebhookReconciler reconciles a NamespacedValidatingWebhook object
type NamespacedValidatingWebhookReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webhook.zoetrope.github.io,resources=namespacedvalidatingwebhooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webhook.zoetrope.github.io,resources=namespacedvalidatingwebhooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webhook.zoetrope.github.io,resources=namespacedvalidatingwebhooks/finalizers,verbs=update
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfiguration,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespacedValidatingWebhook object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *NamespacedValidatingWebhookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var nvw webhookv1.NamespacedValidatingWebhook
	err := r.Get(ctx, req.NamespacedName, &nvw)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if nvw.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&nvw, Finalizer) {
			controllerutil.AddFinalizer(&nvw, Finalizer)
			err = r.Update(ctx, &nvw)
			if err != nil {
				return ctrl.Result{}, nil
			}
		}
	} else {
		logger.Info("starting finalization")
		if err := r.finalize(ctx, nvw); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to finalize: %w", err)
		}
		logger.Info("finished finalization")
		return ctrl.Result{}, nil
	}
	err = r.reconcileWebhookConfiguration(ctx, nvw)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NamespacedValidatingWebhookReconciler) finalize(ctx context.Context, nvw webhookv1.NamespacedValidatingWebhook) error {
	if !controllerutil.ContainsFinalizer(&nvw, Finalizer) {
		return nil
	}

	logger := log.FromContext(ctx)

	config := &admissionv1.ValidatingWebhookConfiguration{}
	if err := r.Get(ctx, client.ObjectKey{Name: nvw.ConfigName()}, config); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		goto CLEANUP
	}

	{
		ownerNamespace := nvw.Labels[LabelOwnerNamespace]
		ownerName := nvw.Labels[LabelOwnerName]
		if ownerNamespace != nvw.Namespace || ownerName != nvw.Name {
			logger.Info("finalization: ignored non-owned ValidatingWebhookConfiguration", "ownerNamespace", ownerNamespace, "ownerName", ownerName)
			goto CLEANUP
		}
	}

	if err := r.Delete(ctx, config); err != nil {
		return fmt.Errorf("failed to delete ValidatingWebhookConfiguration %s: %w", nvw.ConfigName(), err)
	}

	logger.Info("deleted ValidatingWebhookConfiguration", "name", nvw.ConfigName())

CLEANUP:
	controllerutil.RemoveFinalizer(&nvw, Finalizer)
	return r.Update(ctx, &nvw)
}

func (r *NamespacedValidatingWebhookReconciler) reconcileWebhookConfiguration(ctx context.Context, nvw webhookv1.NamespacedValidatingWebhook) error {
	logger := log.FromContext(ctx)

	config := admissionv1apply.ValidatingWebhookConfiguration(nvw.ConfigName()).
		WithLabels(map[string]string{
			LabelCreatedBy: NamespacedValidatingWebhookControllerName,
		})

	webhooks := make([]*admissionv1apply.ValidatingWebhookApplyConfiguration, 0)
	for _, hook := range nvw.Webhooks {
		webhook := admissionv1apply.ValidatingWebhook().
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
		webhook.WithNamespaceSelector(metav1apply.LabelSelector().
			WithMatchExpressions(metav1apply.LabelSelectorRequirement().
				WithKey(corev1.LabelMetadataName).
				WithOperator(metav1.LabelSelectorOpIn).
				WithValues(nvw.Namespace),
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

	var current admissionv1.ValidatingWebhookConfiguration
	err = r.Get(ctx, client.ObjectKey{Name: nvw.ConfigName()}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := admissionv1apply.ExtractValidatingWebhookConfiguration(&current, NamespacedValidatingWebhookControllerName)
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(config, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: NamespacedValidatingWebhookControllerName,
		Force:        pointer.Bool(true),
	})
	if err != nil {
		logger.Error(err, "unable to create or update ValidatingWebhookConfiguration")
		return err
	}

	logger.Info("reconcile ValidatingWebhookConfiguration successfully")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespacedValidatingWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	cfgHandler := func(obj client.Object, q workqueue.RateLimitingInterface) {
		ns := obj.GetLabels()[LabelOwnerNamespace]
		if ns == "" {
			return
		}
		name := obj.GetLabels()[LabelOwnerName]
		if name == "" {
			return
		}
		q.Add(reconcile.Request{NamespacedName: client.ObjectKey{
			Namespace: ns,
			Name:      name,
		}})
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&webhookv1.NamespacedValidatingWebhook{}).
		Watches(&source.Kind{Type: &admissionv1.ValidatingWebhookConfiguration{}}, handler.Funcs{
			UpdateFunc: func(ev event.UpdateEvent, q workqueue.RateLimitingInterface) {
				if ev.ObjectNew.GetDeletionTimestamp() != nil {
					return
				}
				cfgHandler(ev.ObjectNew, q)
			},
			DeleteFunc: func(ev event.DeleteEvent, q workqueue.RateLimitingInterface) {
				cfgHandler(ev.Object, q)
			},
		}).
		Complete(r)
}
