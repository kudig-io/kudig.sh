package resources

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HealthChecker inspects cluster nodes and pods to derive an overall
// cluster health status.
type HealthChecker struct {
	clientset *kubernetes.Clientset
}

// NewHealthChecker creates a new HealthChecker backed by the given clientset.
func NewHealthChecker(clientset *kubernetes.Clientset) *HealthChecker {
	return &HealthChecker{clientset: clientset}
}

// NodeHealth captures the health of a single Kubernetes node.
type NodeHealth struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Conditions map[string]string `json:"conditions"`
}

// PodHealthSummary provides aggregate pod counts across the cluster.
type PodHealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
}

// ClusterHealth is the result of a full cluster health check.
type ClusterHealth struct {
	Nodes     []NodeHealth      `json:"nodes"`
	Pods      PodHealthSummary  `json:"pods"`
	Overall   string            `json:"overall"`
	Timestamp time.Time         `json:"timestamp"`
}

// CheckClusterHealth evaluates all nodes and pods to produce a ClusterHealth
// report. The Overall field is set to "Healthy", "Degraded", or "Unhealthy"
// based on the ratio of ready nodes and running pods.
func (hc *HealthChecker) CheckClusterHealth(ctx context.Context) (*ClusterHealth, error) {
	// -- Nodes --
	nodes, err := hc.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	nodeHealths := make([]NodeHealth, 0, len(nodes.Items))
	readyNodes := 0

	for _, node := range nodes.Items {
		conds := make(map[string]string)
		nodeReady := false

		for _, c := range node.Status.Conditions {
			conds[string(c.Type)] = string(c.Status)
			if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
				nodeReady = true
			}
		}

		status := "NotReady"
		if nodeReady {
			status = "Ready"
			readyNodes++
		}

		nodeHealths = append(nodeHealths, NodeHealth{
			Name:       node.Name,
			Status:     status,
			Conditions: conds,
		})
	}

	// -- Pods --
	pods, err := hc.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	podSummary := PodHealthSummary{Total: len(pods.Items)}
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded {
			podSummary.Healthy++
		} else {
			podSummary.Unhealthy++
		}
	}

	// -- Overall status --
	totalNodes := len(nodes.Items)
	overall := "Healthy"

	if totalNodes == 0 {
		overall = "Unhealthy"
	} else {
		nodeRatio := float64(readyNodes) / float64(totalNodes)
		podRatio := 0.0
		if podSummary.Total > 0 {
			podRatio = float64(podSummary.Healthy) / float64(podSummary.Total)
		}

		if nodeRatio < 0.5 || (podSummary.Total > 0 && podRatio < 0.5) {
			overall = "Unhealthy"
		} else if nodeRatio < 1.0 || (podSummary.Total > 0 && podRatio < 0.9) {
			overall = "Degraded"
		}
	}

	return &ClusterHealth{
		Nodes:     nodeHealths,
		Pods:      podSummary,
		Overall:   overall,
		Timestamp: time.Now().UTC(),
	}, nil
}
