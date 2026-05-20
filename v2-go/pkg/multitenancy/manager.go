package multitenancy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Manager handles multi-tenant lifecycle and resource management.
type Manager struct {
	clientset *kubernetes.Clientset
	tenants   []Tenant
	users     []TenantUser
	mu        sync.RWMutex
	dataDir   string
}

// NewManager creates a new multi-tenancy manager.
func NewManager(clientset *kubernetes.Clientset, dataDir string) *Manager {
	return &Manager{
		clientset: clientset,
		dataDir:   dataDir,
	}
}

// LoadTenants reads tenant and user data from disk.
func (m *Manager) LoadTenants() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	tenantsFile := filepath.Join(m.dataDir, "tenants.json")
	if data, err := os.ReadFile(tenantsFile); err == nil {
		var tenants []Tenant
		if err := json.Unmarshal(data, &tenants); err != nil {
			return fmt.Errorf("unmarshal tenants: %w", err)
		}
		m.tenants = tenants
	}

	usersFile := filepath.Join(m.dataDir, "users.json")
	if data, err := os.ReadFile(usersFile); err == nil {
		var users []TenantUser
		if err := json.Unmarshal(data, &users); err != nil {
			return fmt.Errorf("unmarshal users: %w", err)
		}
		m.users = users
	}

	return nil
}

// SaveTenants persists tenant and user data to disk.
func (m *Manager) SaveTenants() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := os.MkdirAll(m.dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	tenantsFile := filepath.Join(m.dataDir, "tenants.json")
	data, err := json.MarshalIndent(m.tenants, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tenants: %w", err)
	}
	if err := os.WriteFile(tenantsFile, data, 0o644); err != nil {
		return fmt.Errorf("write tenants: %w", err)
	}

	usersFile := filepath.Join(m.dataDir, "users.json")
	data, err = json.MarshalIndent(m.users, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal users: %w", err)
	}
	if err := os.WriteFile(usersFile, data, 0o644); err != nil {
		return fmt.Errorf("write users: %w", err)
	}

	return nil
}

// CreateTenant creates a tenant with its Kubernetes resources.
func (m *Manager) CreateTenant(tenant Tenant) (*Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant.ID = uuid.New().String()
	tenant.CreatedAt = time.Now().UTC()
	tenant.UpdatedAt = tenant.CreatedAt

	if len(tenant.Namespaces) == 0 {
		tenant.Namespaces = []string{fmt.Sprintf("tenant-%s", tenant.Name)}
	}

	// Create namespaces
	for _, ns := range tenant.Namespaces {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Labels: map[string]string{
					"kudig.io/tenant":   tenant.ID,
					"kudig.io/managed":  "true",
				},
			},
		}
		if _, err := m.clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{}); err != nil {
			return nil, fmt.Errorf("create namespace %s: %w", ns, err)
		}
	}

	// Create ResourceQuotas
	if tenant.ResourceQuotas != (ResourceQuotaSpec{}) {
		for _, ns := range tenant.Namespaces {
			quota := buildResourceQuota(ns, tenant.ResourceQuotas)
			if _, err := m.clientset.CoreV1().ResourceQuotas(ns).Create(context.TODO(), quota, metav1.CreateOptions{}); err != nil {
				return nil, fmt.Errorf("create resource quota in %s: %w", ns, err)
			}
		}
	}

	// Create NetworkPolicies
	if tenant.NetworkPolicies.Enabled {
		for _, ns := range tenant.Namespaces {
			if tenant.NetworkPolicies.DefaultDeny {
				policy := buildDefaultDenyPolicy(ns)
				if _, err := m.clientset.NetworkingV1().NetworkPolicies(ns).Create(context.TODO(), policy, metav1.CreateOptions{}); err != nil {
					return nil, fmt.Errorf("create network policy in %s: %w", ns, err)
				}
			}
		}
	}

	// Create RBAC resources
	if tenant.RBAC.Enabled {
		roleName := fmt.Sprintf("kudig-tenant-%s", tenant.ID)
		if tenant.RBAC.DefaultRole != "" {
			roleName = tenant.RBAC.DefaultRole
		}
		for _, ns := range tenant.Namespaces {
			role := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName,
					Namespace: ns,
					Labels:    map[string]string{"kudig.io/tenant": tenant.ID},
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods", "services", "configmaps", "secrets", "persistentvolumeclaims"},
						Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
					},
					{
						APIGroups: []string{"apps"},
						Resources: []string{"deployments", "statefulsets", "replicasets"},
						Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
					},
				},
			}
			if _, err := m.clientset.RbacV1().Roles(ns).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
				return nil, fmt.Errorf("create role in %s: %w", ns, err)
			}
		}
	}

	m.tenants = append(m.tenants, tenant)
	return &tenant, nil
}

// UpdateTenant modifies an existing tenant.
func (m *Manager) UpdateTenant(id string, updates Tenant) (*Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, t := range m.tenants {
		if t.ID == id {
			if updates.Name != "" {
				m.tenants[i].Name = updates.Name
			}
			if updates.Description != "" {
				m.tenants[i].Description = updates.Description
			}
			if updates.ResourceQuotas != (ResourceQuotaSpec{}) {
				m.tenants[i].ResourceQuotas = updates.ResourceQuotas
			}
			m.tenants[i].NetworkPolicies = updates.NetworkPolicies
			m.tenants[i].RBAC = updates.RBAC
			m.tenants[i].UpdatedAt = time.Now().UTC()
			result := m.tenants[i]
			return &result, nil
		}
	}
	return nil, fmt.Errorf("tenant %s not found", id)
}

// DeleteTenant removes a tenant and its Kubernetes resources.
func (m *Manager) DeleteTenant(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var tenant *Tenant
	for i, t := range m.tenants {
		if t.ID == id {
			tenant = &m.tenants[i]
			break
		}
	}
	if tenant == nil {
		return fmt.Errorf("tenant %s not found", id)
	}

	// Delete namespaces (cascading delete removes quotas, policies, roles)
	for _, ns := range tenant.Namespaces {
		if err := m.clientset.CoreV1().Namespaces().Delete(context.TODO(), ns, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("delete namespace %s: %w", ns, err)
		}
	}

	// Remove from slice
	for i, t := range m.tenants {
		if t.ID == id {
			m.tenants = append(m.tenants[:i], m.tenants[i+1:]...)
			break
		}
	}

	// Remove associated users
	var remaining []TenantUser
	for _, u := range m.users {
		if u.TenantID != id {
			remaining = append(remaining, u)
		}
	}
	m.users = remaining

	return nil
}

// GetTenant returns a tenant by ID.
func (m *Manager) GetTenant(id string) *Tenant {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.tenants {
		if t.ID == id {
			result := t
			return &result
		}
	}
	return nil
}

// ListTenants returns tenants matching the filter.
func (m *Manager) ListTenants(filter TenantFilter) []Tenant {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Tenant
	for _, t := range m.tenants {
		if filter.Name != "" && !strings.Contains(strings.ToLower(t.Name), strings.ToLower(filter.Name)) {
			continue
		}
		if filter.HasNamespace != "" {
			found := false
			for _, ns := range t.Namespaces {
				if ns == filter.HasNamespace {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, t)
	}
	return result
}

// GetTenantUsage queries actual resource usage for a tenant.
func (m *Manager) GetTenantUsage(ctx context.Context, id string) (*TenantUsage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tenant *Tenant
	for _, t := range m.tenants {
		if t.ID == id {
			tenant = &t
			break
		}
	}
	if tenant == nil {
		return nil, fmt.Errorf("tenant %s not found", id)
	}

	usage := &TenantUsage{
		TenantID:   tenant.ID,
		TenantName: tenant.Name,
		Namespaces: make(map[string]NamespaceUsage),
		Quotas:     tenant.ResourceQuotas,
	}

	for _, ns := range tenant.Namespaces {
		var nsUsage NamespaceUsage

		pods, err := m.clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			nsUsage.Pods = len(pods.Items)
		}

		svcs, err := m.clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			nsUsage.Services = len(svcs.Items)
		}

		pvcs, err := m.clientset.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			nsUsage.PVCs = len(pvcs.Items)
		}

		for _, pod := range pods.Items {
			for _, c := range pod.Spec.Containers {
				if req, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
					nsUsage.CPURequests += float64(req.MilliValue()) / 1000.0
				}
				if lim, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
					nsUsage.CPULimits += float64(lim.MilliValue()) / 1000.0
				}
				if req, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
					nsUsage.MemoryRequests += float64(req.Value()) / (1024 * 1024 * 1024)
				}
				if lim, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
					nsUsage.MemoryLimits += float64(lim.Value()) / (1024 * 1024 * 1024)
				}
			}
		}

		usage.Namespaces[ns] = nsUsage
		usage.Totals.Pods += nsUsage.Pods
		usage.Totals.Services += nsUsage.Services
		usage.Totals.PVCs += nsUsage.PVCs
		usage.Totals.CPURequests += nsUsage.CPURequests
		usage.Totals.CPULimits += nsUsage.CPULimits
		usage.Totals.MemoryRequests += nsUsage.MemoryRequests
		usage.Totals.MemoryLimits += nsUsage.MemoryLimits
	}

	return usage, nil
}

// AddUser adds a user to a tenant.
func (m *Manager) AddUser(user TenantUser) (*TenantUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify tenant exists
	found := false
	for _, t := range m.tenants {
		if t.ID == user.TenantID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("tenant %s not found", user.TenantID)
	}

	user.ID = uuid.New().String()
	user.CreatedAt = time.Now().UTC()

	m.users = append(m.users, user)
	return &user, nil
}

// UpdateUser modifies an existing user.
func (m *Manager) UpdateUser(id string, updates TenantUser) (*TenantUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, u := range m.users {
		if u.ID == id {
			if updates.Username != "" {
				m.users[i].Username = updates.Username
			}
			if updates.Email != "" {
				m.users[i].Email = updates.Email
			}
			if updates.Role != "" {
				m.users[i].Role = updates.Role
			}
			if len(updates.Namespaces) > 0 {
				m.users[i].Namespaces = updates.Namespaces
			}
			result := m.users[i]
			return &result, nil
		}
	}
	return nil, fmt.Errorf("user %s not found", id)
}

// DeleteUser removes a user.
func (m *Manager) DeleteUser(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, u := range m.users {
		if u.ID == id {
			m.users = append(m.users[:i], m.users[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("user %s not found", id)
}

// ListUsers returns users matching the filter.
func (m *Manager) ListUsers(filter UserFilter) []TenantUser {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []TenantUser
	for _, u := range m.users {
		if filter.TenantID != "" && u.TenantID != filter.TenantID {
			continue
		}
		if filter.Role != "" && u.Role != filter.Role {
			continue
		}
		if filter.Username != "" && !strings.Contains(strings.ToLower(u.Username), strings.ToLower(filter.Username)) {
			continue
		}
		result = append(result, u)
	}
	return result
}

// GetUser returns a user by ID.
func (m *Manager) GetUser(id string) *TenantUser {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if u.ID == id {
			result := u
			return &result
		}
	}
	return nil
}

// getDefaultTenants returns a set of default tenants for new installations.
func (m *Manager) getDefaultTenants() []Tenant {
	return []Tenant{
		{
			Name:        "default",
			Description: "Default tenant for general workloads",
			Namespaces:  []string{"default"},
			ResourceQuotas: ResourceQuotaSpec{
				CPU:      "4",
				Memory:   "8Gi",
				Pods:     "50",
				Services: "20",
				PVCs:     "10",
			},
			NetworkPolicies: NetworkPolicySpec{Enabled: false, DefaultDeny: false},
			RBAC:            RBACSpec{Enabled: true, DefaultRole: "tenant-member"},
		},
	}
}

// buildResourceQuota creates a Kubernetes ResourceQuota spec.
func buildResourceQuota(namespace string, spec ResourceQuotaSpec) *corev1.ResourceQuota {
	hard := corev1.ResourceList{}
	if spec.CPU != "" {
		hard[corev1.ResourceRequestsCPU] = resource.MustParse(spec.CPU)
		hard[corev1.ResourceLimitsCPU] = resource.MustParse(spec.CPU)
	}
	if spec.Memory != "" {
		hard[corev1.ResourceRequestsMemory] = resource.MustParse(spec.Memory)
		hard[corev1.ResourceLimitsMemory] = resource.MustParse(spec.Memory)
	}
	if spec.Pods != "" {
		hard[corev1.ResourcePods] = resource.MustParse(spec.Pods)
	}
	if spec.Services != "" {
		hard[corev1.ResourceServices] = resource.MustParse(spec.Services)
	}
	if spec.PVCs != "" {
		hard[corev1.ResourcePersistentVolumeClaims] = resource.MustParse(spec.PVCs)
	}

	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kudig-quota",
			Namespace: namespace,
			Labels:    map[string]string{"kudig.io/managed": "true"},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: hard,
		},
	}
}

// buildDefaultDenyPolicy creates a default-deny ingress network policy.
func buildDefaultDenyPolicy(namespace string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kudig-default-deny",
			Namespace: namespace,
			Labels:    map[string]string{"kudig.io/managed": "true"},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}
}
