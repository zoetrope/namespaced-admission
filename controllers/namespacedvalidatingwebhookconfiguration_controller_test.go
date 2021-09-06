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
	_ "embed"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	webhookv1 "github.com/zoetrope/namespaced-admission/api/v1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed testdata/validating.yaml
var validatingYAML []byte

var _ = Describe("NamespacedValidatingWebhookConfiguration controller", func() {
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(k8sCfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ToNot(HaveOccurred())

		nr := &NamespacedValidatingWebhookConfigurationReconciler{
			Client:         mgr.GetClient(),
			Scheme:         scheme,
			TargetLabelKey: "target",
		}
		err = nr.SetupWithManager(mgr)
		Expect(err).ToNot(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	It("should create and delete ValidatingWebhookConfiguration", func() {

		nmw := &webhookv1.NamespacedValidatingWebhookConfiguration{}
		err := yaml.Unmarshal(validatingYAML, nmw)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Create(ctx, nmw)
		Expect(err).NotTo(HaveOccurred())

		var config *admissionv1.ValidatingWebhookConfiguration
		Eventually(func() error {
			config = &admissionv1.ValidatingWebhookConfiguration{}
			return k8sClient.Get(ctx, client.ObjectKey{Name: nmw.ConfigName()}, config)
		}).Should(Succeed())

		Expect(config.Webhooks).Should(HaveLen(1))
		Expect(config.Webhooks[0].NamespaceSelector).ShouldNot(BeNil())
		Expect(config.Webhooks[0].NamespaceSelector.MatchExpressions).Should(HaveLen(1))
		Expect(config.Webhooks[0].NamespaceSelector.MatchExpressions[0]).Should(
			MatchAllFields(Fields{
				"Key":      Equal("target"),
				"Operator": Equal(metav1.LabelSelectorOpIn),
				"Values":   Equal([]string{"foo"}),
			}))

		err = k8sClient.Delete(ctx, nmw)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() bool {
			config = &admissionv1.ValidatingWebhookConfiguration{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: nmw.ConfigName()}, config)
			if err != nil {
				return apierrors.IsNotFound(err)
			}
			return config.DeletionTimestamp != nil
		}).Should(BeTrue())
	})
})
