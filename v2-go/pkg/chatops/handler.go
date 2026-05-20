package chatops

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Handler handles ChatOps commands against a Kubernetes cluster.
type Handler struct {
	clientset *kubernetes.Clientset
}

// NewHandler creates a new command Handler with the given Kubernetes clientset.
func NewHandler(clientset *kubernetes.Clientset) *Handler {
	return &Handler{clientset: clientset}
}

// HandleCommand parses and executes a chatops command, returning a markdown result.
func (h *Handler) HandleCommand(ctx context.Context, command string) (string, error) {
	parts := strings.Fields(strings.TrimSpace(command))
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	switch strings.ToLower(parts[0]) {
	case "cluster":
		return h.handleCluster(ctx)
	case "deployment":
		ns := "default"
		if len(parts) > 1 {
			ns = parts[1]
		}
		return h.handleDeployment(ctx, ns)
	case "pod":
		ns := "default"
		if len(parts) > 1 {
			ns = parts[1]
		}
		return h.handlePod(ctx, ns)
	case "node":
		return h.handleNode(ctx)
	case "service":
		ns := "default"
		if len(parts) > 1 {
			ns = parts[1]
		}
		return h.handleService(ctx, ns)
	case "help":
		return h.handleHelp(), nil
	default:
		return "", fmt.Errorf("unknown command: %s", parts[0])
	}
}

func (h *Handler) handleCluster(ctx context.Context) (string, error) {
	nodes, err := h.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("## Cluster Overview\n\n")
	sb.WriteString(fmt.Sprintf("**Total Nodes:** %d\n\n", len(nodes.Items)))
	sb.WriteString("| Node | Status | Roles | Version |\n")
	sb.WriteString("|------|--------|-------|---------|\n")

	for _, node := range nodes.Items {
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				status = "Ready"
				break
			}
		}

		roles := []string{}
		for label := range node.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {
				role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
				if role != "" {
					roles = append(roles, role)
				}
			}
		}
		if len(roles) == 0 {
			roles = append(roles, "<none>")
		}

		version := node.Status.NodeInfo.KubeletVersion
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			node.Name, status, strings.Join(roles, ","), version))
	}

	return sb.String(), nil
}

func (h *Handler) handleDeployment(ctx context.Context, namespace string) (string, error) {
	deployments, err := h.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list deployments in %s: %w", namespace, err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Deployments in `%s`\n\n", namespace))
	sb.WriteString("| Name | Ready | Up-to-date | Available | Age |\n")
	sb.WriteString("|------|-------|------------|-----------|-----|\n")

	for _, dep := range deployments.Items {
		ready := dep.Status.ReadyReplicas

		var desired int32
		if dep.Spec.Replicas != nil {
			desired = *dep.Spec.Replicas
		}

		age := formatAge(dep.CreationTimestamp.Time)

		sb.WriteString(fmt.Sprintf("| %s | %d/%d | %d | %d | %s |\n",
			dep.Name, ready, desired, dep.Status.UpdatedReplicas, dep.Status.AvailableReplicas, age))
	}

	if len(deployments.Items) == 0 {
		sb.WriteString("| *(none)* | | | | |\n")
	}

	return sb.String(), nil
}

func (h *Handler) handlePod(ctx context.Context, namespace string) (string, error) {
	pods, err := h.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list pods in %s: %w", namespace, err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Pods in `%s`\n\n", namespace))
	sb.WriteString("| Name | Status | Restarts | Age |\n")
	sb.WriteString("|------|--------|----------|-----|\n")

	for _, pod := range pods.Items {
		status := string(pod.Status.Phase)
		var restarts int32
		for _, cs := range pod.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}

		age := formatAge(pod.CreationTimestamp.Time)

		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s |\n",
			pod.Name, status, restarts, age))
	}

	if len(pods.Items) == 0 {
		sb.WriteString("| *(none)* | | | |\n")
	}

	return sb.String(), nil
}

func (h *Handler) handleNode(ctx context.Context) (string, error) {
	nodes, err := h.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("## Node Details\n\n")

	for _, node := range nodes.Items {
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				status = "Ready"
				break
			}
		}

		cpu := node.Status.Capacity.Cpu()
		mem := node.Status.Capacity.Memory()
		sb.WriteString(fmt.Sprintf("### %s\n", node.Name))
		sb.WriteString(fmt.Sprintf("- **Status:** %s\n", status))
		sb.WriteString(fmt.Sprintf("- **CPU:** %s\n", cpu.String()))
		sb.WriteString(fmt.Sprintf("- **Memory:** %s\n", mem.String()))
		sb.WriteString(fmt.Sprintf("- **OS:** %s %s\n",
			node.Status.NodeInfo.OperatingSystem, node.Status.NodeInfo.OSImage))
		sb.WriteString(fmt.Sprintf("- **Kernel:** %s\n", node.Status.NodeInfo.KernelVersion))
		sb.WriteString(fmt.Sprintf("- **Container Runtime:** %s\n\n", node.Status.NodeInfo.ContainerRuntimeVersion))
	}

	return sb.String(), nil
}

func (h *Handler) handleService(ctx context.Context, namespace string) (string, error) {
	services, err := h.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list services in %s: %w", namespace, err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Services in `%s`\n\n", namespace))
	sb.WriteString("| Name | Type | Cluster-IP | External-IP | Ports |\n")
	sb.WriteString("|------|------|------------|-------------|-------|\n")

	for _, svc := range services.Items {
		externalIP := "<none>"
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			externalIP = svc.Status.LoadBalancer.Ingress[0].IP
		}

		ports := []string{}
		for _, p := range svc.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}
		if len(ports) == 0 {
			ports = append(ports, "<none>")
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			svc.Name, svc.Spec.Type, svc.Spec.ClusterIP, externalIP, strings.Join(ports, ",")))
	}

	if len(services.Items) == 0 {
		sb.WriteString("| *(none)* | | | | |\n")
	}

	return sb.String(), nil
}

func (h *Handler) handleHelp() string {
	return `## ChatOps Commands

| Command | Description |
|---------|-------------|
| ` + "`cluster`" + ` | Show cluster overview (nodes, status, versions) |
| ` + "`deployment [namespace]`" + ` | List deployments (default namespace) |
| ` + "`pod [namespace]`" + ` | List pods (default namespace) |
| ` + "`node`" + ` | Show detailed node information |
| ` + "`service [namespace]`" + ` | List services (default namespace) |
| ` + "`help`" + ` | Show this help message |
`
}

// formatAge returns a human-readable duration string for a given creation time.
func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
