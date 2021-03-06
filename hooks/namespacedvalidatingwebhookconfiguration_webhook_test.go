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
	_ "embed"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/zoetrope/namespaced-admission/api/v1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed testdata/validating.yaml
var validatingYAML []byte

//go:embed testdata/invalid-validating.yaml
var invalidValidatingYAML []byte

var _ = Describe("NamespacedValidatingWebhookConfiguration webhook", func() {
	ctx := context.Background()

	It("should mutate scope", func() {
		nmw := &v1.NamespacedValidatingWebhookConfiguration{}
		err := yaml.Unmarshal(validatingYAML, nmw)
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.Create(ctx, nmw)
		Expect(err).NotTo(HaveOccurred())

		var mutated *v1.NamespacedValidatingWebhookConfiguration
		Eventually(func() error {
			mutated = &v1.NamespacedValidatingWebhookConfiguration{}
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

	It("should deny creation of NamespacedValidatingWebhookConfiguration that contains a forbidden rule", func() {
		nmw := &v1.NamespacedValidatingWebhookConfiguration{}
		err := yaml.Unmarshal(invalidValidatingYAML, nmw)
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.Create(ctx, nmw)
		Expect(err).To(HaveOccurred())
	})
})
