package rbacanalysis

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAnalyzer_AnalyzeRBAC(t *testing.T) {
	client := fake.NewClientset(
		// Roles
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "role-a", Namespace: "ns1"},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "role-b", Namespace: "ns1"},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "role-c", Namespace: "ns2"},
		},
		// ClusterRoles
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "system:node"},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "system:discovery"},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "admin"},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "view"},
		},
		// RoleBindings
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "rb-a", Namespace: "ns1"},
			RoleRef:    rbacv1.RoleRef{Kind: "Role", Name: "role-a"},
			Subjects: []rbacv1.Subject{
				{Kind: "ServiceAccount", Name: "sa1", Namespace: "ns1"},
			},
		},
		// ClusterRoleBindings
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "crb-a"},
			RoleRef:    rbacv1.RoleRef{Kind: "ClusterRole", Name: "system:node"},
			Subjects: []rbacv1.Subject{
				{Kind: "Group", Name: "system:nodes"},
			},
		},
	)

	a := NewAnalyzer(client)
	result, err := a.AnalyzeRBAC(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalRoles != 3 {
		t.Errorf("expected 3 roles, got %d", result.TotalRoles)
	}
	if result.TotalClusterRoles != 4 {
		t.Errorf("expected 4 cluster roles, got %d", result.TotalClusterRoles)
	}
	if result.TotalBindings != 1 {
		t.Errorf("expected 1 binding, got %d", result.TotalBindings)
	}
	if result.TotalClusterBindings != 1 {
		t.Errorf("expected 1 cluster binding, got %d", result.TotalClusterBindings)
	}

	// RolesByNamespace
	if len(result.RolesByNamespace["ns1"]) != 2 {
		t.Errorf("expected 2 roles in ns1, got %d", len(result.RolesByNamespace["ns1"]))
	}
	if len(result.RolesByNamespace["ns2"]) != 1 {
		t.Errorf("expected 1 role in ns2, got %d", len(result.RolesByNamespace["ns2"]))
	}

	// ClusterRolesByPrefix
	if len(result.ClusterRolesByPrefix["system"]) != 2 {
		t.Errorf("expected 2 cluster roles with prefix 'system', got %d", len(result.ClusterRolesByPrefix["system"]))
	}
	if len(result.ClusterRolesByPrefix["admin"]) != 1 {
		t.Errorf("expected 1 cluster role with prefix 'admin', got %d", len(result.ClusterRolesByPrefix["admin"]))
	}

	// BindingsBySubject
	if len(result.BindingsBySubject) != 2 {
		t.Errorf("expected 2 subjects, got %d", len(result.BindingsBySubject))
	}

	// BindingsByRole
	if len(result.BindingsByRole) != 2 {
		t.Errorf("expected 2 role entries, got %d", len(result.BindingsByRole))
	}
}

func TestAnalyzer_ListRoles(t *testing.T) {
	client := fake.NewClientset(
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "default"}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "r2", Namespace: "default"}},
	)
	a := NewAnalyzer(client)
	roles, err := a.ListRoles(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}

func TestAnalyzer_ListClusterRoles(t *testing.T) {
	client := fake.NewClientset(
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "cr1"}},
	)
	a := NewAnalyzer(client)
	crs, err := a.ListClusterRoles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(crs) != 1 {
		t.Errorf("expected 1 cluster role, got %d", len(crs))
	}
}
