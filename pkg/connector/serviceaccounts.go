package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-openshift/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type serviceAccountBuilder struct {
	client *client.Client
}

func (o *serviceAccountBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return serviceAccountResourceType
}

func (o *serviceAccountBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	svcAccounts, err := o.client.K8sClient.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, "", nil, err
	}
	resources := make([]*v2.Resource, 0, len(svcAccounts.Items))
	for _, svcAccount := range svcAccounts.Items {
		if svcAccount.Name == "builder" || svcAccount.Name == "deployer" || svcAccount.Name == "default" {
			continue
		}
		profile := map[string]interface{}{
			"name": svcAccount.Name,
		}

		name := fmt.Sprintf("%s:%s", svcAccount.Namespace, svcAccount.Name)
		resource, err := rs.NewUserResource(
			name,
			serviceAccountResourceType,
			name,
			[]rs.UserTraitOption{
				rs.WithUserProfile(profile),
				rs.WithAccountType(v2.UserTrait_ACCOUNT_TYPE_SYSTEM),
			},
			rs.WithDescription(fmt.Sprintf("%s Kubernetes ServiceAccount in %s", svcAccount.Name, svcAccount.Namespace)),
		)
		if err != nil {
			return nil, "", nil, err
		}
		resources = append(resources, resource)
	}
	return resources, "", nil, nil
}

func (o *serviceAccountBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *serviceAccountBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newServiceAccountBuilder(clt *client.Client) *serviceAccountBuilder {
	return &serviceAccountBuilder{client: clt}
}
