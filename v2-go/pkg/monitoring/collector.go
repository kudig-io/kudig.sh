package monitoring

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Collector gathers Kubernetes cluster metrics by querying the API server.
type Collector struct {
	clientset kubernetes.Interface
}

// NewCollector creates a new metrics Collector wrapping the given clientset.
func NewCollector(clientset kubernetes.Interface) *Collector {
	return &Collector{clientset: clientset}
}

// CollectClusterMetrics gathers a full snapshot of cluster metrics.
func (c *Collector) CollectClusterMetrics(ctx context.Context, clusterName string) (*ClusterMetrics, error) {
	metrics := &ClusterMetrics{
		Timestamp:   time.Now().UTC(),
		ClusterName: clusterName,
	}

	// Collect nodes
	nodeDetails, err := c.collectNodeMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect node metrics: %w", err)
	}
	metrics.Nodes = NodeMetricsSummary{
		Total:   len(nodeDetails),
		Details: nodeDetails,
	}
	for _, nd := range nodeDetails {
		switch nd.Status {
		case "Ready":
			metrics.Nodes.Ready++
			metrics.ReadyNodes++
		default:
			metrics.Nodes.NotReady++
		}
		if nd.Schedulable {
			metrics.Nodes.Schedulable++
		} else {
			metrics.Nodes.Unschedulable++
		}
	}
	metrics.TotalNodes = len(nodeDetails)

	// Collect pods
	podDetails, err := c.collectPodMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect pod metrics: %w", err)
	}
	metrics.Pods = PodMetricsSummary{
		Total:   len(podDetails),
		Details: podDetails,
	}
	for _, pd := range podDetails {
		switch pd.Phase {
		case "Running":
			metrics.Pods.Running++
			metrics.RunningPods++
		case "Pending":
			metrics.Pods.Pending++
			metrics.PendingPods++
		case "Succeeded":
			metrics.Pods.Succeeded++
		case "Failed":
			metrics.Pods.Failed++
			metrics.FailedPods++
		default:
			metrics.Pods.Unknown++
		}
	}
	metrics.TotalPods = len(podDetails)

	// Collect resource metrics (requests/limits aggregation)
	resMetrics, err := c.collectResourceMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect resource metrics: %w", err)
	}
	metrics.Resources = *resMetrics

	// Collect event metrics
	evtMetrics, err := c.collectEventMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect event metrics: %w", err)
	}
	metrics.EventMetrics = *evtMetrics

	// Collect counts for other resource types
	nsList, _ := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if nsList != nil {
		metrics.Namespaces = len(nsList.Items)
	}
	deployList, _ := c.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if deployList != nil {
		metrics.Deployments = len(deployList.Items)
	}
	svcList, _ := c.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if svcList != nil {
		metrics.Services = len(svcList.Items)
	}

	return metrics, nil
}

// collectNodeMetrics queries each node for its status and conditions.
func (c *Collector) collectNodeMetrics(ctx context.Context) ([]NodeDetail, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	details := make([]NodeDetail, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		nd := NodeDetail{
			Name:           node.Name,
			CPUCapacity:    formatQuantity(node.Status.Capacity.Cpu()),
			MemoryCapacity: formatQuantity(node.Status.Capacity.Memory()),
			PodCount:       int(node.Status.Capacity.Pods().Value()),
			Conditions:     make(map[string]string),
			Labels:         node.Labels,
		}

		for _, cond := range node.Status.Conditions {
			nd.Conditions[string(cond.Type)] = string(cond.Status)
			if cond.Type == corev1.NodeReady {
				if cond.Status == corev1.ConditionTrue {
					nd.Status = "Ready"
				} else {
					nd.Status = "NotReady"
				}
			}
		}

		// Schedulability
		nd.Schedulable = !node.Spec.Unschedulable

		details = append(details, nd)
	}
	return details, nil
}

// collectPodMetrics queries all pods across namespaces for status info.
func (c *Collector) collectPodMetrics(ctx context.Context) ([]PodDetail, error) {
	pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	details := make([]PodDetail, 0, len(pods.Items))
	for _, pod := range pods.Items {
		pd := PodDetail{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     string(pod.Status.Phase),
			Node:      pod.Spec.NodeName,
		}

		for _, cs := range pod.Status.ContainerStatuses {
			pd.Restarts += cs.RestartCount
		}

		details = append(details, pd)
	}
	return details, nil
}

// collectResourceMetrics aggregates CPU/memory requests and limits across all pods.
func (c *Collector) collectResourceMetrics(ctx context.Context) (*ResourceMetrics, error) {
	rm := &ResourceMetrics{}

	podList, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return rm, err
	}

	var cpuRequests, cpuLimits int64
	var memRequests, memLimits int64

	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if req, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				cpuRequests += req.MilliValue()
			}
			if lim, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
				cpuLimits += lim.MilliValue()
			}
			if req, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				memRequests += req.Value()
			}
			if lim, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
				memLimits += lim.Value()
			}
		}
	}

	rm.CPURequests = fmt.Sprintf("%dm", cpuRequests)
	rm.CPULimits = fmt.Sprintf("%dm", cpuLimits)
	rm.MemoryRequests = formatMemoryFromBytes(memRequests)
	rm.MemoryLimits = formatMemoryFromBytes(memLimits)

	return rm, nil
}

// collectEventMetrics gathers event statistics across the cluster.
func (c *Collector) collectEventMetrics(ctx context.Context) (*EventMetric, error) {
	em := &EventMetric{}

	events, err := c.clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return em, err
	}

	em.Total = len(events.Items)
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	for _, evt := range events.Items {
		switch evt.Type {
		case "Warning":
			em.Warnings++
		case "Normal":
			em.Normal++
		}
		if evt.LastTimestamp.After(oneHourAgo) || evt.CreationTimestamp.After(oneHourAgo) {
			em.LastHour++
		}
	}

	return em, nil
}

// formatQuantity formats a resource.Quantity pointer to a human-readable string.
func formatQuantity(q *resource.Quantity) string {
	if q == nil {
		return "0"
	}
	return q.String()
}

// formatMemoryFromBytes formats a byte count into a human-readable string.
func formatMemoryFromBytes(bytes int64) string {
	const (
		KiB = 1024
		MiB = 1024 * KiB
		GiB = 1024 * MiB
	)
	switch {
	case bytes >= GiB:
		return fmt.Sprintf("%.2fGi", float64(bytes)/float64(GiB))
	case bytes >= MiB:
		return fmt.Sprintf("%.2fMi", float64(bytes)/float64(MiB))
	case bytes >= KiB:
		return fmt.Sprintf("%.2fKi", float64(bytes)/float64(KiB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
