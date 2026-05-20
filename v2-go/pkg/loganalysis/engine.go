package loganalysis

import (
	"context"
	"fmt"
	"io"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Engine provides log-analysis operations backed by a Kubernetes API client.
type Engine struct {
	clientset *kubernetes.Clientset
	analyzer  *Analyzer
}

// NewEngine returns a new Engine that uses the supplied Kubernetes clientset.
func NewEngine(clientset *kubernetes.Clientset) *Engine {
	return &Engine{
		clientset: clientset,
		analyzer:  NewAnalyzer(),
	}
}

// GetPodLogs fetches the last tailLines of logs from the specified container.
func (e *Engine) GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64) (string, error) {
	opts := &corev1.PodLogOptions{
		Container: container,
		TailLines: &tailLines,
	}
	req := e.clientset.CoreV1().Pods(namespace).GetLogs(name, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to open log stream for %s/%s: %w", namespace, name, err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return "", fmt.Errorf("failed to read log stream for %s/%s: %w", namespace, name, err)
	}
	return string(data), nil
}

// AnalyzePodLogs fetches pod logs and runs a full analysis. The default tail
// window is 1000 lines.
func (e *Engine) AnalyzePodLogs(ctx context.Context, namespace, name, container string) (*LogAnalysis, error) {
	const defaultTail int64 = 1000

	// Verify the pod exists and optionally resolve an empty container name.
	pod, err := e.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}
	if container == "" && len(pod.Spec.Containers) > 0 {
		container = pod.Spec.Containers[0].Name
	}

	// Determine tail lines from pod annotation if present, otherwise default.
	tail := defaultTail
	if v, ok := pod.Annotations["kudig.io/log-tail-lines"]; ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			tail = n
		}
	}

	logs, err := e.GetPodLogs(ctx, namespace, name, container, tail)
	if err != nil {
		return nil, err
	}

	return e.analyzer.AnalyzeLogs(logs), nil
}
