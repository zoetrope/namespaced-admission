[![CI](https://github.com/zoetrope/namespaced-admission/actions/workflows/ci.yaml/badge.svg)](https://github.com/zoetrope/namespaced-admission/actions/workflows/ci.yaml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/zoetrope/namespaced-admission?tab=overview)](https://pkg.go.dev/github.com/zoetrope/namespaced-admission?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/zoetrope/namespaced-admission)](https://goreportcard.com/report/github.com/zoetrope/namespaced-admission)

# namespaced-admission

namespaced-admission is a Kubernetes controller that allows tenant users to deploy [Admission Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).

## Concepts

In order to deploy AdmissionWebhook, we need to create [ValidatingWebhookConfiguration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#validatingwebhookconfiguration-v1-admissionregistration-k8s-io) or [MutatingWebhookConfiguration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#mutatingwebhookconfiguration-v1-admissionregistration-k8s-io) resources.
However, they are cluster-scoped resources, so tenant users cannot create them.

namespaced-admission provides namespace-scoped Custom Resources `NamespacedValidatingWebhookConfiguration` and `NamespacedMutatingWebhookConfiguration`.
namespaced-admission can safely create `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` from these resources.

These namespace-scoped resources are almost the same as cluster-scoped resources, but with the following limitations.

- Cannot specify `namespaceSelector`.
- The resources listed in `rules` must be accessible via the target webhook server. 
- The resources listed in `rules` must have a scope of `Namespaced`. 

`namespaceSelector` will be automatically filled with conditions that match the "kubernetes.io/metadata.name" label assigned to the namespace of the custom resource.
("kubernetes.io/metadata.name" label is supported in Kubernetes 1.21 and later)

You can change the label key by using `--target-label-key` option.
Please note the following to set this option up. ([Accurate](https://github.com/cybozu-go/accurate) will help you)

- Tenant users should not be able to modify namespace resources.
- Namespaces with the same label specified in `--target-label-key` option must be guaranteed to have the same permissions.

namespaced-admission uses `serviceAccountName` filed to verify whether the resources listed in `rules` are accessible or not.
Then apply the ServiceAccount to the target webhook.
See [Role](./config/dummy/role.yaml), [RoleBinding](./config/dummy/role_binding.yaml) and [ServiceAccount](./config/dummy/service_account.yaml).

## Demo

1. Prepare Docker, kubectl, [kind (Kubernetes-In-Docker)](https://kind.sigs.k8s.io/)
2. Launch a Kubernetes cluster with kind.
```
kind create cluster
```
3. Deploy cert-manager.
```
make deploy-cert-manager
```
4. Deploy namespaced-admission
```
make install
make docker-build
make docker-load
make deploy
```
5. Deploy admission webhook server for a tenant user.
```
make docker-build-dummy
make docker-load-dummy
make deploy-dummy
```
