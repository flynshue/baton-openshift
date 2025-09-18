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

type roleBindingBuilder struct {
	client *client.Client
}

func (o *roleBindingBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return roleBindingResourceType
}

func (o *roleBindingBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	rbList, err := o.client.K8sClient.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, "", nil, err
	}
	resources := make([]*v2.Resource, 0, len(rbList.Items))
	for _, rb := range rbList.Items {
		if strings.Contains(rb.Name, "system:") || rb.Namespace == "openshift-console-user-settings" || strings.Contains(rb.RoleRef.Name, "system:") {
			continue
		}
		r, err := convertRole2Resource(rb)
		if err != nil {
			return nil, "", nil, err
		}
		resources = append(resources, r)
	}
	return resources, "", nil, nil
}

func (o *roleBindingBuilder) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var entitlements []*v2.Entitlement
	namespace := strings.Split(resource.DisplayName, ":")[0]
	entitlementName := fmt.Sprintf("%s:%s", namespace, "member")
	nsEnt := ent.NewAssignmentEntitlement(
		resource,
		entitlementName,
		ent.WithDisplayName(fmt.Sprintf("\"%s\" RoleBinding Member in \"%s\" namespace", resource.DisplayName, namespace)),
		ent.WithDescription(fmt.Sprintf("Grants membership to the \"%s\" rolebinding in namespace \"%s\"", resource.DisplayName, namespace)),
		ent.WithGrantableTo(
			userResourceType,
			groupResourceType,
			serviceAccountResourceType,
		),
	)
	entitlements = append(entitlements, nsEnt)
	return entitlements, "", nil, nil
}

func (o *roleBindingBuilder) Grants(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	namespace, rolebindingName := parseResourceName(resource.DisplayName)
	rb, err := o.client.K8sClient.RbacV1().RoleBindings(namespace).Get(ctx, rolebindingName, metav1.GetOptions{})
	if err != nil {
		return nil, "", nil, err
	}
	grants := make([]*v2.Grant, 0, len(rb.Subjects))
	for _, subject := range rb.Subjects {
		entitlementName := fmt.Sprintf("%s:%s", namespace, "member")
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
}

func newroleBindingBuilder(client *client.Client) *roleBindingBuilder {
	return &roleBindingBuilder{client: client}
}

func convertRole2Resource(roleBinding rbacv1.RoleBinding) (*v2.Resource, error) {
	resourceName := fmt.Sprintf("%s:%s:%s", roleBinding.Namespace, roleBinding.Name, roleBinding.RoleRef.Name)
	profile := map[string]interface{}{
		"name": resourceName,
		"uid":  string(roleBinding.UID),
	}
	resource, err := rs.NewRoleResource(
		resourceName,
		roleBindingResourceType,
		resourceName,
		[]rs.RoleTraitOption{rs.WithRoleProfile(profile)},
		rs.WithDescription(fmt.Sprintf("Rolebinding %s grants %s %s in namespace %s", roleBinding.Name, roleBinding.RoleRef.Kind, roleBinding.RoleRef.Name, roleBinding.Namespace)),
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func parseResourceName(resourceName string) (namespace, name string) {
	namespace, resource, _ := strings.Cut(resourceName, ":")
	if strings.Count(resource, ":") > 1 {
		fields := strings.Split(resource, ":")
		name = fields[0] + ":" + fields[1]
		return
	}
	name, _, _ = strings.Cut(resource, ":")
	return
}
