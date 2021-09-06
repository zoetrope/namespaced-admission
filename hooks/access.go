package hooks

import (
	"context"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authv1client "k8s.io/client-go/kubernetes/typed/authorization/v1"
)

func canAccess(ctx context.Context, authClient authv1client.AuthorizationV1Interface, verb, group, version, resource, ns string) (bool, error) {
	sar := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: ns,
				Verb:      verb,
				Group:     group,
				Version:   version,
				Resource:  resource,
			},
		},
	}
	res, err := authClient.SelfSubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
	if err != nil {
		return false, err
	}
	return res.Status.Allowed, nil
}
