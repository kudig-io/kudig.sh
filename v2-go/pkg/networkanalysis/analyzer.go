// Package networkanalysis provides Kubernetes network resource analysis.
package networkanalysis

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Analyzer performs network analysis against a Kubernetes cluster.
type Analyzer struct {
	clientset *kubernetes.Clientset
}

// NewAnalyzer creates a new network Analyzer.
func NewAnalyzer(clientset *kubernetes.Clientset) *Analyzer {
	return &Analyzer{clientset: clientset}
}

// ExposedService describes a service that exposes ports externally.
type ExposedService struct {
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
	Type      string              `json:"type"`
	Ports     []corev1.ServicePort `json:"ports"`
}

// NetworkAnalysis holds the result of a full network analysis.
type NetworkAnalysis struct {
	TotalNetworkPolicies int                 `json:"total_network_policies"`
	TotalServices        int                 `json:"total_services"`
	TotalIngresses       int                 `json:"total_ingresses"`
	PoliciesByNamespace  map[string][]string `json:"policies_by_namespace"`
	ServicesByType       map[string][]string `json:"services_by_type"`
	IngressesByHost      map[string][]string `json:"ingresses_by_host"`
	ExposedServices      []ExposedService    `json:"exposed_services"`
	Timestamp            time.Time           `json:"timestamp"`
}

// ListNetworkPolicies returns all NetworkPolicies in the given namespace.
func (a *Analyzer) ListNetworkPolicies(ctx context.Context, ns string) ([]networkingv1.NetworkPolicy, error) {
	list, err := a.clientset.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListIngressClasses returns all IngressClasses in the cluster.
func (a *Analyzer) ListIngressClasses(ctx context.Context) ([]networkingv1.IngressClass, error) {
	list, err := a.clientset.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// AnalyzeNetwork collects NetworkPolicies, Services, and Ingresses across all
// namespaces and produces a NetworkAnalysis.
func (a *Analyzer) AnalyzeNetwork(ctx context.Context) (*NetworkAnalysis, error) {
	policies, err := a.ListNetworkPolicies(ctx, "")
	if err != nil {
		return nil, err
	}

	services, err := a.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ingresses, err := a.clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	analysis := &NetworkAnalysis{
		TotalNetworkPolicies: len(policies),
		TotalServices:        len(services.Items),
		TotalIngresses:       len(ingresses.Items),
		PoliciesByNamespace:  make(map[string][]string),
		ServicesByType:       make(map[string][]string),
		IngressesByHost:      make(map[string][]string),
		ExposedServices:      make([]ExposedService, 0),
		Timestamp:            time.Now(),
	}

	// Index policies by namespace.
	for _, p := range policies {
		ns := p.Namespace
		analysis.PoliciesByNamespace[ns] = append(analysis.PoliciesByNamespace[ns], p.Name)
	}

	// Index services by type and collect exposed services.
	for _, svc := range services.Items {
		typeStr := string(svc.Spec.Type)
		analysis.ServicesByType[typeStr] = append(analysis.ServicesByType[typeStr], svc.Name)

		if isExposed(svc.Spec.Type) {
			analysis.ExposedServices = append(analysis.ExposedServices, ExposedService{
				Name:      svc.Name,
				Namespace: svc.Namespace,
				Type:      typeStr,
				Ports:     svc.Spec.Ports,
			})
		}
	}

	// Index ingresses by host.
	for _, ing := range ingresses.Items {
		for _, rule := range ing.Spec.Rules {
			host := rule.Host
			if host == "" {
				host = "*"
			}
			analysis.IngressesByHost[host] = append(analysis.IngressesByHost[host], ing.Name)
		}
	}

	return analysis, nil
}

// isExposed returns true when a service type is externally accessible.
func isExposed(t corev1.ServiceType) bool {
	return t == corev1.ServiceTypeLoadBalancer || t == corev1.ServiceTypeNodePort
}
