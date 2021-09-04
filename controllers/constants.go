package controllers

const MetaPrefix = "namespaced-webhook.zoetrope.github.io/"

const (
	LabelCreatedBy      = "app.kubernetes.io/created-by"
	LabelOwnerNamespace = MetaPrefix + "owner-namespace"
	LabelOwnerName      = MetaPrefix + "owner-name"
)

const Finalizer = MetaPrefix + "finalizer"

// Label or annotation values
const (
	NamespacedMutatingWebhookControllerName   = "namespaced-mutating-webhook-controller"
	NamespacedValidatingWebhookControllerName = "namespaced-validating-webhook-controller"
)
