package multitenancy

import "time"

// Tenant represents an isolated multi-tenant environment.
type Tenant struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Namespaces       []string          `json:"namespaces"`
	ResourceQuotas   ResourceQuotaSpec `json:"resourceQuotas"`
	NetworkPolicies  NetworkPolicySpec `json:"networkPolicies"`
	RBAC             RBACSpec          `json:"rbac"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

// ResourceQuotaSpec defines resource limits for a tenant.
type ResourceQuotaSpec struct {
	CPU      string `json:"cpu"`
	Memory   string `json:"memory"`
	Pods     string `json:"pods"`
	Services string `json:"services"`
	PVCs     string `json:"pvcs"`
}

// NetworkPolicySpec defines network policy settings for a tenant.
type NetworkPolicySpec struct {
	Enabled     bool `json:"enabled"`
	DefaultDeny bool `json:"defaultDeny"`
}

// RBACSpec defines RBAC settings for a tenant.
type RBACSpec struct {
	Enabled     bool   `json:"enabled"`
	DefaultRole string `json:"defaultRole"`
}

// TenantUser represents a user within a tenant.
type TenantUser struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenantId"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	Namespaces []string  `json:"namespaces"`
	CreatedAt  time.Time `json:"createdAt"`
}

// TenantUsage tracks actual resource usage per tenant.
type TenantUsage struct {
	TenantID   string                    `json:"tenantId"`
	TenantName string                    `json:"tenantName"`
	Namespaces map[string]NamespaceUsage `json:"namespaces"`
	Totals     UsageSummary              `json:"totals"`
	Quotas     ResourceQuotaSpec         `json:"quotas"`
}

// NamespaceUsage holds resource usage for a single namespace.
type NamespaceUsage struct {
	Pods            int     `json:"pods"`
	Services        int     `json:"services"`
	PVCs            int     `json:"pvcs"`
	CPURequests     float64 `json:"cpuRequests"`
	CPULimits       float64 `json:"cpuLimits"`
	MemoryRequests  float64 `json:"memoryRequests"`
	MemoryLimits    float64 `json:"memoryLimits"`
}

// UsageSummary is an aggregate of usage across namespaces.
type UsageSummary struct {
	Pods            int     `json:"pods"`
	Services        int     `json:"services"`
	PVCs            int     `json:"pvcs"`
	CPURequests     float64 `json:"cpuRequests"`
	CPULimits       float64 `json:"cpuLimits"`
	MemoryRequests  float64 `json:"memoryRequests"`
	MemoryLimits    float64 `json:"memoryLimits"`
}

// TenantFilter is used to filter tenant listings.
type TenantFilter struct {
	Name       string `json:"name,omitempty"`
	HasNamespace string `json:"hasNamespace,omitempty"`
}

// UserFilter is used to filter user listings.
type UserFilter struct {
	TenantID string `json:"tenantId,omitempty"`
	Role     string `json:"role,omitempty"`
	Username string `json:"username,omitempty"`
}
