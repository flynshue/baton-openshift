package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-openshift/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type groupBuilder struct {
	namespace string
	client    *client.Client
}

func (o *groupBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return groupResourceType
}

func (o *groupBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	groups, err := o.client.ListGroups(ctx)
	if err != nil {
		return nil, "", nil, err
	}
	return groups, "", nil, nil
}

func (o *groupBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	assigmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(userResourceType),
		ent.WithDisplayName(fmt.Sprintf("%s Team member", resource.DisplayName)),
		ent.WithDescription(fmt.Sprintf("Access to %s team", resource.DisplayName)),
	}

	rv = append(rv, ent.NewAssignmentEntitlement(
		resource,
		"member",
		assigmentOptions...,
	))

	return rv, "", nil, nil
}

func (o *groupBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	group, err := o.client.UsersClient.Groups().Get(ctx, resource.DisplayName, metav1.GetOptions{})
	if err != nil {
		return nil, "", nil, err
	}
	users := group.Users
	grants := make([]*v2.Grant, 0, len(users))
	for _, user := range users {
		principal := &v2.Resource{Id: &v2.ResourceId{ResourceType: userResourceType.Id, Resource: user}}
		grants = append(grants, grant.NewGrant(resource, "member", principal))
	}
	return grants, "", nil, nil
}

func newGroupBuilder(namespace string, clt *client.Client) *groupBuilder {
	return &groupBuilder{
		namespace: namespace,
		client:    clt,
	}
}
