package constants

const MetaPrefix = "namespaced-webhook.zoetrope.github.io/"

const (
	LabelCreatedBy      = "app.kubernetes.io/created-by"
	LabelOwnerNamespace = MetaPrefix + "owner-namespace"
	LabelOwnerName      = MetaPrefix + "owner-name"
)

const Finalizer = MetaPrefix + "finalizer"

// Label or annotation values
const (
	NamespacedMutatingWebhookConfigurationControllerName   = "namespaced-mutating-webhook-controller"
	NamespacedValidatingWebhookConfigurationControllerName = "namespaced-validating-webhook-controller"
)
