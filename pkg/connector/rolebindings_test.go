package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetResourceName(t *testing.T) {
	testCases := []struct {
		name         string
		resourceName string
		namespace    string
	}{
		{
			name:         "vsphere-cloud-controller-manager:vsphere-cloud-controller-manager",
			resourceName: "openshift-cloud-controller-manager:vsphere-cloud-controller-manager:vsphere-cloud-controller-manager:vsphere-cloud-controller-manager",
			namespace:    "openshift-cloud-controller-manager",
		},
		{
			name:         "foo-rolebinding",
			resourceName: "foo-namespace:foo-rolebinding:foo-role",
			namespace:    "foo-namespace",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			namespace, name := parseResourceName(tc.resourceName)
			assert.Equal(t, tc.name, name, "rolebinding name should match")
			assert.Equal(t, tc.namespace, namespace, "namespace name should match")
		})
	}
}

func TestParseClusterRoleResource(t *testing.T) {
	testCases := []struct {
		name         string
		resourceName string
		role         string
	}{
		{
			name:         "vsphere-cloud-controller-manager:vsphere-cloud-controller-manager",
			resourceName: "vsphere-cloud-controller-manager:vsphere-cloud-controller-manager:ClusterRole:vsphere-cloud-controller-manager-role",
			role:         "vsphere-cloud-controller-manager-role",
		},
		{
			name:         "foo-rolebinding",
			resourceName: "foo-rolebinding:ClusterRole:foo-role",
			role:         "foo-role",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name, role := parseClusterRoleResource(tc.resourceName)
			assert.Equal(t, tc.name, name, "resource name should match")
			assert.Equal(t, tc.role, role, "role name should match")
		})
	}
}
