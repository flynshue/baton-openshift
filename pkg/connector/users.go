package connector

import (
	"context"

	"github.com/conductorone/baton-openshift/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type userBuilder struct {
	namespace string
	client    *client.Client
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

func (o *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	groupList, err := o.client.UsersClient.Groups().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, "", nil, err
	}
	users := make([]string, 0, len(groupList.Items))
	for _, group := range groupList.Items {
		userList := group.Users
		users = append(users, userList...)
	}
	resources := make([]*v2.Resource, 0, len(users))
	for _, user := range users {
		profile := map[string]interface{}{
			"name": user,
		}

		resource, err := rs.NewUserResource(
			user,
			userResourceType,
			user,
			[]rs.UserTraitOption{
				rs.WithUserProfile(profile),
			},
		)
		if err != nil {
			return nil, "", nil, err
		}
		resources = append(resources, resource)
	}
	return resources, "", nil, nil
}

func (o *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *userBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newUserBuilder(namespace string, clt *client.Client) *userBuilder {
	return &userBuilder{
		namespace: namespace,
		client:    clt,
	}
}
