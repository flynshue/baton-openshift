package connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/conductorone/baton-openshift/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterRoleBindingBuilder struct {
	client *client.Client
}

func (o *clusterRoleBindingBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return clusterRoleBindingResourceType
}

func (o *clusterRoleBindingBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	rbList, err := o.client.K8sClient.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, "", nil, err
	}
	resources := make([]*v2.Resource, 0, len(rbList.Items))
	for _, rb := range rbList.Items {
		if strings.Contains(rb.Name, "system:") || strings.Contains(rb.RoleRef.Name, "system:") {
			continue
		}
		r, err := convertClusterRole2Resource(rb)
		if err != nil {
			return nil, "", nil, err
		}
		resources = append(resources, r)
	}
	return resources, "", nil, nil
}

func (o *clusterRoleBindingBuilder) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var entitlements []*v2.Entitlement
	name, role := parseClusterRoleResource(resource.DisplayName)
	entitlementName := fmt.Sprintf("%s:%s", name, "member")
	nsEnt := ent.NewAssignmentEntitlement(
		resource,
		entitlementName,
		ent.WithDisplayName(fmt.Sprintf("\"%s\" ClusterRoleBinding Member", name)),
		ent.WithDescription(fmt.Sprintf("Grants membership to the \"%s\" ClusterRoleBinding with ClusterRole %s", name, role)),
		ent.WithGrantableTo(
			userResourceType,
			groupResourceType,
			serviceAccountResourceType,
		),
	)
	entitlements = append(entitlements, nsEnt)
	return entitlements, "", nil, nil
}

func (o *clusterRoleBindingBuilder) Grants(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	// rb, err := o.client.K8sClient.RbacV1().RoleBindings(namespace).Get(ctx, rolebindingName, metav1.GetOptions{})
	name, _ := parseClusterRoleResource(resource.DisplayName)
	rb, err := o.client.K8sClient.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, "", nil, err
	}
	grants := make([]*v2.Grant, 0, len(rb.Subjects))
	for _, subject := range rb.Subjects {
		entitlementName := fmt.Sprintf("%s:%s", name, "member")
		principal := &v2.Resource{Id: &v2.ResourceId{Resource: subject.Name}}
		switch subject.Kind {
		case rbacv1.GroupKind:
			principal.Id.ResourceType = groupResourceType.Id
			g := grant.NewGrant(
				resource,
				entitlementName,
				principal,
				grant.WithAnnotation(&v2.GrantExpandable{EntitlementIds: []string{
					fmt.Sprintf("%s:%s:member", groupResourceType.Id, subject.Name),
				}}),
			)
			grants = append(grants, g)
		case rbacv1.UserKind:
			principal.Id.ResourceType = userResourceType.Id
			g := grant.NewGrant(
				resource,
				entitlementName,
				principal,
			)
			grants = append(grants, g)
		case rbacv1.ServiceAccountKind:
			if subject.Name == "builder" || subject.Name == "deployer" || subject.Name == "default" {
				continue
			}
			principal.Id.ResourceType = serviceAccountResourceType.Id
			principal.Id.Resource = fmt.Sprintf("%s:%s", subject.Namespace, subject.Name)
			g := grant.NewGrant(
				resource,
				entitlementName,
				principal,
			)
			grants = append(grants, g)
		default:
			continue
		}

	}
	return grants, "", nil, nil
	// return nil, "", nil, nil
}

func newClusterRoleBindingBuilder(client *client.Client) *clusterRoleBindingBuilder {
	return &clusterRoleBindingBuilder{client: client}
}

func convertClusterRole2Resource(roleBinding rbacv1.ClusterRoleBinding) (*v2.Resource, error) {
	resourceName := fmt.Sprintf("%s:ClusterRole:%s", roleBinding.Name, roleBinding.RoleRef.Name)
	profile := map[string]interface{}{
		"name": resourceName,
		"uid":  string(roleBinding.UID),
	}
	resource, err := rs.NewRoleResource(
		resourceName,
		clusterRoleBindingResourceType,
		resourceName,
		[]rs.RoleTraitOption{rs.WithRoleProfile(profile)},
		rs.WithDescription(fmt.Sprintf("ClusterRoleBinding %s grants %s/%s", roleBinding.Name, roleBinding.RoleRef.Kind, roleBinding.RoleRef.Name)),
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func parseClusterRoleResource(resourceName string) (name, role string) {
	name, role, _ = strings.Cut(resourceName, ":ClusterRole:")
	return
}
