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
	"context"
	_ "embed"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed testdata/mutating.yaml
var mutatingYAML []byte

var _ = Describe("NamespacedMutatingWebhookConfiguration webhook", func() {
	ctx := context.Background()

	It("should mutate scope", func() {
		ns := &corev1.Namespace{}
		ns.Name = "test3"
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		nmw := &NamespacedMutatingWebhookConfiguration{}
		err = yaml.Unmarshal(mutatingYAML, nmw)
		Expect(err).NotTo(HaveOccurred())

		err = k8sClient.Create(ctx, nmw)
		Expect(err).NotTo(HaveOccurred())

		var mutated *NamespacedMutatingWebhookConfiguration
		Eventually(func() error {
			mutated = &NamespacedMutatingWebhookConfiguration{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: nmw.Namespace, Name: nmw.Name}, mutated)
		}).Should(Succeed())

		Expect(mutated.Webhooks).ShouldNot(BeEmpty())
		for _, hook := range mutated.Webhooks {
			Expect(hook.Rules).ShouldNot(BeEmpty())
			for _, rule := range hook.Rules {
				Expect(rule.Scope).ShouldNot(BeNil())
				Expect(*rule.Scope).Should(Equal(admissionv1.NamespacedScope))
			}
		}
	})
})
