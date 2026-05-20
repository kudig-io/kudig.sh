// Package rbacanalysis provides Kubernetes RBAC resource analysis.
package rbacanalysis

import (
	"context"
	"fmt"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Analyzer performs RBAC analysis against a Kubernetes cluster.
type Analyzer struct {
	clientset *kubernetes.Clientset
}

// NewAnalyzer creates a new RBAC Analyzer.
func NewAnalyzer(clientset *kubernetes.Clientset) *Analyzer {
	return &Analyzer{clientset: clientset}
}

// SubjectBinding describes a single binding of a subject to a role.
type SubjectBinding struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Role      string `json:"role"`
	RoleKind  string `json:"role_kind"`
}

// RoleBindingInfo describes a role and the subjects bound to it.
type RoleBindingInfo struct {
	Type      string   `json:"type"`
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Subjects  []string `json:"subjects"`
}

// RBACAnalysis holds the result of a full RBAC analysis.
type RBACAnalysis struct {
	TotalRoles          int                              `json:"total_roles"`
	TotalClusterRoles   int                              `json:"total_cluster_roles"`
	TotalBindings       int                              `json:"total_bindings"`
	TotalClusterBindings int                             `json:"total_cluster_bindings"`
	RolesByNamespace    map[string][]string              `json:"roles_by_namespace"`
	BindingsBySubject   map[string][]SubjectBinding      `json:"bindings_by_subject"`
	BindingsByRole      map[string][]RoleBindingInfo     `json:"bindings_by_role"`
	Timestamp           time.Time                        `json:"timestamp"`
}

// ListRoles returns all Roles in the given namespace.
func (a *Analyzer) ListRoles(ctx context.Context, ns string) ([]rbacv1.Role, error) {
	list, err := a.clientset.RbacV1().Roles(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListClusterRoles returns all ClusterRoles in the cluster.
func (a *Analyzer) ListClusterRoles(ctx context.Context) ([]rbacv1.ClusterRole, error) {
	list, err := a.clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// AnalyzeRBAC collects Roles, ClusterRoles, RoleBindings, and
// ClusterRoleBindings, then indexes them by subject and by role.
func (a *Analyzer) AnalyzeRBAC(ctx context.Context) (*RBACAnalysis, error) {
	roles, err := a.ListRoles(ctx, "")
	if err != nil {
		return nil, err
	}

	clusterRoles, err := a.ListClusterRoles(ctx)
	if err != nil {
		return nil, err
	}

	roleBindings, err := a.clientset.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterRoleBindings, err := a.clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	analysis := &RBACAnalysis{
		TotalRoles:           len(roles),
		TotalClusterRoles:    len(clusterRoles),
		TotalBindings:        len(roleBindings.Items),
		TotalClusterBindings: len(clusterRoleBindings.Items),
		RolesByNamespace:     make(map[string][]string),
		BindingsBySubject:    make(map[string][]SubjectBinding),
		BindingsByRole:       make(map[string][]RoleBindingInfo),
		Timestamp:            time.Now(),
	}

	// Index roles by namespace.
	for _, r := range roles {
		analysis.RolesByNamespace[r.Namespace] = append(analysis.RolesByNamespace[r.Namespace], r.Name)
	}

	// Process namespace-scoped RoleBindings.
	for _, rb := range roleBindings.Items {
		roleName := rb.RoleRef.Name
		roleKind := rb.RoleRef.Kind
		bindingKey := fmt.Sprintf("%s/%s", rb.Namespace, roleName)

		info := RoleBindingInfo{
			Type:      "RoleBinding",
			Name:      rb.Name,
			Namespace: rb.Namespace,
			Subjects:  make([]string, 0, len(rb.Subjects)),
		}

		for _, s := range rb.Subjects {
			subjectKey := fmt.Sprintf("%s:%s:%s", s.Kind, s.Namespace, s.Name)
			info.Subjects = append(info.Subjects, subjectKey)

			analysis.BindingsBySubject[subjectKey] = append(analysis.BindingsBySubject[subjectKey], SubjectBinding{
				Type:      "RoleBinding",
				Name:      rb.Name,
				Namespace: rb.Namespace,
				Role:      roleName,
				RoleKind:  roleKind,
			})
		}

		analysis.BindingsByRole[bindingKey] = append(analysis.BindingsByRole[bindingKey], info)
	}

	// Process cluster-scoped ClusterRoleBindings.
	for _, crb := range clusterRoleBindings.Items {
		roleName := crb.RoleRef.Name
		roleKind := crb.RoleRef.Kind
		bindingKey := fmt.Sprintf("(cluster)/%s", roleName)

		info := RoleBindingInfo{
			Type:      "ClusterRoleBinding",
			Name:      crb.Name,
			Namespace: "",
			Subjects:  make([]string, 0, len(crb.Subjects)),
		}

		for _, s := range crb.Subjects {
			subjectKey := fmt.Sprintf("%s:%s:%s", s.Kind, s.Namespace, s.Name)
			info.Subjects = append(info.Subjects, subjectKey)

			analysis.BindingsBySubject[subjectKey] = append(analysis.BindingsBySubject[subjectKey], SubjectBinding{
				Type:      "ClusterRoleBinding",
				Name:      crb.Name,
				Namespace: "",
				Role:      roleName,
				RoleKind:  roleKind,
			})
		}

		analysis.BindingsByRole[bindingKey] = append(analysis.BindingsByRole[bindingKey], info)
	}

	return analysis, nil
}
