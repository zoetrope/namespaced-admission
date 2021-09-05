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
	"github.com/zoetrope/namespaced-webhook/pkg/constants"
	"github.com/zoetrope/namespaced-webhook/pkg/utils"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

// NamespacedMutatingWebhookConfigurationReconciler reconciles a NamespacedMutatingWebhookConfiguration object
type NamespacedMutatingWebhookConfigurationReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	TargetLabelKey string
}

//+kubebuilder:rbac:groups=admissionregistration.zoetrope.github.io,resources=namespacedmutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=admissionregistration.zoetrope.github.io,resources=namespacedmutatingwebhookconfigurations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=admissionregistration.zoetrope.github.io,resources=namespacedmutatingwebhookconfigurations/finalizers,verbs=update
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespacedMutatingWebhookConfiguration object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *NamespacedMutatingWebhookConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var nmw webhookv1.NamespacedMutatingWebhookConfiguration
	err := r.Get(ctx, req.NamespacedName, &nmw)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if nmw.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&nmw, constants.Finalizer) {
			controllerutil.AddFinalizer(&nmw, constants.Finalizer)
			err = r.Update(ctx, &nmw)
			if err != nil {
				return ctrl.Result{}, nil
			}
		}
	} else {
		logger.Info("starting finalization")
		if err := r.finalize(ctx, nmw); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to finalize: %w", err)
		}
		logger.Info("finished finalization")
		return ctrl.Result{}, nil
	}
	err = r.reconcileWebhookConfiguration(ctx, nmw)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NamespacedMutatingWebhookConfigurationReconciler) finalize(ctx context.Context, nmw webhookv1.NamespacedMutatingWebhookConfiguration) error {
	if !controllerutil.ContainsFinalizer(&nmw, constants.Finalizer) {
		return nil
	}

	logger := log.FromContext(ctx)

	config := &admissionv1.MutatingWebhookConfiguration{}
	if err := r.Get(ctx, client.ObjectKey{Name: nmw.ConfigName()}, config); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		goto CLEANUP
	}

	{
		ownerNamespace := config.Labels[constants.LabelOwnerNamespace]
		ownerName := config.Labels[constants.LabelOwnerName]
		if ownerNamespace != nmw.Namespace || ownerName != nmw.Name {
			logger.Info("finalization: ignored non-owned MutatingWebhookConfiguration", "ownerNamespace", ownerNamespace, "ownerName", ownerName)
			goto CLEANUP
		}
	}

	if err := r.Delete(ctx, config); err != nil {
		return fmt.Errorf("failed to delete MutatingWebhookConfiguration %s: %w", nmw.ConfigName(), err)
	}

	logger.Info("deleted MutatingWebhookConfiguration", "name", nmw.ConfigName())

CLEANUP:
	controllerutil.RemoveFinalizer(&nmw, constants.Finalizer)
	return r.Update(ctx, &nmw)
}

func (r *NamespacedMutatingWebhookConfigurationReconciler) reconcileWebhookConfiguration(ctx context.Context, nmw webhookv1.NamespacedMutatingWebhookConfiguration) error {
	logger := log.FromContext(ctx)

	configName := nmw.Namespace + "-" + nmw.Name

	config := admissionv1apply.MutatingWebhookConfiguration(configName).
		WithLabels(map[string]string{
			constants.LabelCreatedBy:      constants.NamespacedMutatingWebhookConfigurationControllerName,
			constants.LabelOwnerNamespace: nmw.Namespace,
			constants.LabelOwnerName:      nmw.Name,
		})

	config.WithLabels(nmw.Labels)
	config.WithAnnotations(nmw.Annotations)

	ns := &corev1.Namespace{}
	err := r.Get(ctx, client.ObjectKey{Name: nmw.Namespace}, ns)
	if err != nil {
		return err
	}
	labelValue := ns.Labels[r.TargetLabelKey]
	if labelValue == "" {
		return fmt.Errorf("namespace '%s' does not have '%s' label", ns.Name, r.TargetLabelKey)
	}

	webhooks := make([]*admissionv1apply.MutatingWebhookApplyConfiguration, 0)
	for _, hook := range nmw.Webhooks {
		webhook := admissionv1apply.MutatingWebhook()
		err := utils.DeepCopy(webhook, hook)
		if err != nil {
			return err
		}
		webhook.WithNamespaceSelector(metav1apply.LabelSelector().
			WithMatchExpressions(metav1apply.LabelSelectorRequirement().
				WithKey(r.TargetLabelKey).
				WithOperator(metav1.LabelSelectorOpIn).
				WithValues(labelValue),
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

	currApplyConfig, err := admissionv1apply.ExtractMutatingWebhookConfiguration(&current, constants.NamespacedMutatingWebhookConfigurationControllerName)
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(config, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: constants.NamespacedMutatingWebhookConfigurationControllerName,
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
func (r *NamespacedMutatingWebhookConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	cfgHandler := func(obj client.Object, q workqueue.RateLimitingInterface) {
		ns := obj.GetLabels()[constants.LabelOwnerNamespace]
		if ns == "" {
			return
		}
		name := obj.GetLabels()[constants.LabelOwnerName]
		if name == "" {
			return
		}
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: ns,
			Name:      name,
		}})
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&webhookv1.NamespacedMutatingWebhookConfiguration{}).
		Watches(&source.Kind{Type: &admissionv1.MutatingWebhookConfiguration{}}, handler.Funcs{
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
