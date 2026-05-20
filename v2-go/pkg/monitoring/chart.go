package monitoring

import (
	"fmt"
	"strings"
)

// Chart generates terminal-based bar chart visualizations of monitoring data.
type Chart struct {
	barChar  rune
	maxWidth int
}

// NewChart creates a Chart with sensible defaults.
func NewChart() *Chart {
	return &Chart{
		barChar:  '█',
		maxWidth: 50,
	}
}

// GenerateClusterChart produces an ASCII summary of cluster-level metrics.
func (c *Chart) GenerateClusterChart(metrics ClusterMetrics) string {
	var sb strings.Builder

	sb.WriteString(c.generateSeparator("Cluster Overview"))
	sb.WriteString(fmt.Sprintf("  Cluster:    %s\n", metrics.ClusterName))
	sb.WriteString(fmt.Sprintf("  Timestamp:  %s\n", metrics.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString("\n")

	sb.WriteString(c.generateSeparator("Node Summary"))
	sb.WriteString(fmt.Sprintf("  Total: %d  |  Ready: %d  |  NotReady: %d\n",
		metrics.TotalNodes, metrics.ReadyNodes, metrics.Nodes.NotReady))
	sb.WriteString(c.generateBarChart("  Readiness ", float64(metrics.ReadyNodes), float64(metrics.TotalNodes)))
	sb.WriteString("\n")

	sb.WriteString(c.generateSeparator("Pod Summary"))
	sb.WriteString(fmt.Sprintf("  Total: %d  |  Running: %d  |  Pending: %d  |  Failed: %d\n",
		metrics.TotalPods, metrics.RunningPods, metrics.PendingPods, metrics.FailedPods))
	sb.WriteString(c.generateBarChart("  Running   ", float64(metrics.RunningPods), float64(metrics.TotalPods)))
	sb.WriteString(c.generateBarChart("  Pending   ", float64(metrics.PendingPods), float64(metrics.TotalPods)))
	sb.WriteString(c.generateBarChart("  Failed    ", float64(metrics.FailedPods), float64(metrics.TotalPods)))
	sb.WriteString("\n")

	sb.WriteString(c.generateSeparator("Resources"))
	sb.WriteString(fmt.Sprintf("  CPU Requests:    %s\n", metrics.Resources.CPURequests))
	sb.WriteString(fmt.Sprintf("  CPU Limits:      %s\n", metrics.Resources.CPULimits))
	sb.WriteString(fmt.Sprintf("  Memory Requests: %s\n", metrics.Resources.MemoryRequests))
	sb.WriteString(fmt.Sprintf("  Memory Limits:   %s\n", metrics.Resources.MemoryLimits))
	sb.WriteString("\n")

	sb.WriteString(c.generateSeparator("Events"))
	sb.WriteString(fmt.Sprintf("  Total: %d  |  Warnings: %d  |  Normal: %d  |  Last Hour: %d\n",
		metrics.EventMetrics.Total, metrics.EventMetrics.Warnings,
		metrics.EventMetrics.Normal, metrics.EventMetrics.LastHour))

	return sb.String()
}

// GenerateNodeChart produces an ASCII chart of per-node metrics.
func (c *Chart) GenerateNodeChart(details []NodeDetail) string {
	var sb strings.Builder

	sb.WriteString(c.generateSeparator("Node Details"))

	for _, nd := range details {
		sb.WriteString(fmt.Sprintf("\n  Node: %s\n", nd.Name))
		sb.WriteString(fmt.Sprintf("    Status: %s | Schedulable: %v\n", nd.Status, nd.Schedulable))
		sb.WriteString(fmt.Sprintf("    CPU: %s capacity | Memory: %s capacity\n",
			nd.CPUCapacity, nd.MemoryCapacity))
		sb.WriteString(fmt.Sprintf("    Pods: %d\n", nd.PodCount))

		if len(nd.Conditions) > 0 {
			sb.WriteString("    Conditions:\n")
			for condType, condStatus := range nd.Conditions {
				sb.WriteString(fmt.Sprintf("      %s: %s\n", condType, condStatus))
			}
		}
	}

	return sb.String()
}

// generateBarChart renders a horizontal bar using █ characters.
// The label is left-padded, value/max controls bar width.
func (c *Chart) generateBarChart(label string, value, max float64) string {
	if max <= 0 {
		return fmt.Sprintf("%s %s 0/0\n", label, strings.Repeat(string(c.barChar), 0))
	}

	ratio := value / max
	barLen := int(ratio * float64(c.maxWidth))
	if barLen > c.maxWidth {
		barLen = c.maxWidth
	}
	if barLen < 0 {
		barLen = 0
	}

	pct := ratio * 100
	return fmt.Sprintf("%s %s %.0f/%.0f (%.1f%%)\n",
		label,
		strings.Repeat(string(c.barChar), barLen)+strings.Repeat(" ", c.maxWidth-barLen),
		value, max, pct,
	)
}

// generateSeparator creates a titled section separator line.
func (c *Chart) generateSeparator(title string) string {
	if title == "" {
		return strings.Repeat("─", 60) + "\n"
	}
	return fmt.Sprintf("── %s %s\n", title, strings.Repeat("─", 55-len(title)))
}
