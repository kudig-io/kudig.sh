package web

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Resources wraps a Kubernetes clientset for resource queries.
type Resources struct {
	clientset *kubernetes.Clientset
}

// NewResources creates a Resources from the default kubeconfig.
func NewResources() (*Resources, error) {
	kubeconfig := ""
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = fmt.Sprintf("%s/.kube/config", home)
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	return &Resources{clientset: cs}, nil
}

// NewResourcesFromClientset creates a Resources from an existing clientset.
func NewResourcesFromClientset(cs *kubernetes.Clientset) *Resources {
	return &Resources{clientset: cs}
}

// Clientset returns the underlying Kubernetes clientset.
func (r *Resources) Clientset() *kubernetes.Clientset {
	return r.clientset
}

// --- Namespaces ---

// ListNamespaces returns all namespaces.
func (r *Resources) ListNamespaces() ([]corev1.Namespace, error) {
	ctx := context.Background()
	ns, err := r.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	return ns.Items, nil
}

// --- Pods ---

// ListPods returns pods in the given namespace.
func (r *Resources) ListPods(namespace string) ([]corev1.Pod, error) {
	ctx := context.Background()
	pods, err := r.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods in %s: %w", namespace, err)
	}
	return pods.Items, nil
}

// GetPod returns a single pod by namespace and name.
func (r *Resources) GetPod(namespace, name string) (*corev1.Pod, error) {
	ctx := context.Background()
	pod, err := r.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get pod %s/%s: %w", namespace, name, err)
	}
	return pod, nil
}

// DeletePod deletes a pod by namespace and name.
func (r *Resources) DeletePod(namespace, name string) error {
	ctx := context.Background()
	err := r.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("delete pod %s/%s: %w", namespace, name, err)
	}
	return nil
}

// GetPodLogs returns the logs of a pod container.
func (r *Resources) GetPodLogs(namespace, name string, tailLines int64) (string, error) {
	ctx := context.Background()
	opts := &corev1.PodLogOptions{TailLines: &tailLines}
	req := r.clientset.CoreV1().Pods(namespace).GetLogs(name, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("stream logs %s/%s: %w", namespace, name, err)
	}
	defer stream.Close()
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 32*1024)
	for {
		n, readErr := stream.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if readErr != nil {
			break
		}
	}
	return string(buf), nil
}

// --- Nodes ---

// ListNodes returns all cluster nodes.
func (r *Resources) ListNodes() ([]corev1.Node, error) {
	ctx := context.Background()
	nodes, err := r.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	return nodes.Items, nil
}

// GetNode returns a single node by name.
func (r *Resources) GetNode(name string) (*corev1.Node, error) {
	ctx := context.Background()
	node, err := r.clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get node %s: %w", name, err)
	}
	return node, nil
}

// GetNodeMetrics returns raw metrics-server data for all nodes.
func (r *Resources) GetNodeMetrics() (interface{}, error) {
	ctx := context.Background()
	result := r.clientset.RESTClient().Get().
		AbsPath("/apis/metrics.k8s.io/v1beta1/nodes").
		Do(ctx)
	if result.Error() != nil {
		return nil, fmt.Errorf("get node metrics: %w", result.Error())
	}
	raw, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("read node metrics: %w", err)
	}
	var data interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("decode node metrics: %w", err)
	}
	return data, nil
}

// --- Deployments ---

// ListDeployments returns deployments in the given namespace.
func (r *Resources) ListDeployments(namespace string) (interface{}, error) {
	ctx := context.Background()
	deps, err := r.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments in %s: %w", namespace, err)
	}
	return deps.Items, nil
}

// GetDeployment returns a single deployment by namespace and name.
func (r *Resources) GetDeployment(namespace, name string) (interface{}, error) {
	ctx := context.Background()
	dep, err := r.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get deployment %s/%s: %w", namespace, name, err)
	}
	return dep, nil
}

// ScaleDeployment sets the replica count on a deployment.
func (r *Resources) ScaleDeployment(namespace, name string, replicas int32) error {
	ctx := context.Background()
	scale, err := r.clientset.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get scale %s/%s: %w", namespace, name, err)
	}
	scale.Spec.Replicas = replicas
	_, err = r.clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update scale %s/%s: %w", namespace, name, err)
	}
	return nil
}

// RestartDeployment triggers a rolling restart by updating the pod template annotation.
func (r *Resources) RestartDeployment(namespace, name string) error {
	ctx := context.Background()
	dep, err := r.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get deployment %s/%s: %w", namespace, name, err)
	}
	if dep.Spec.Template.Annotations == nil {
		dep.Spec.Template.Annotations = make(map[string]string)
	}
	dep.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z")
	_, err = r.clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("restart deployment %s/%s: %w", namespace, name, err)
	}
	return nil
}

// GetDeploymentPods returns pods belonging to a deployment via label selector.
func (r *Resources) GetDeploymentPods(namespace, name string) ([]corev1.Pod, error) {
	ctx := context.Background()
	dep, err := r.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get deployment %s/%s: %w", namespace, name, err)
	}
	selector := metav1.FormatLabelSelector(dep.Spec.Selector)
	pods, err := r.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("list pods for deployment %s/%s: %w", namespace, name, err)
	}
	return pods.Items, nil
}

// GetDeploymentStatus returns the replica status of a deployment.
func (r *Resources) GetDeploymentStatus(namespace, name string) (map[string]int32, error) {
	ctx := context.Background()
	dep, err := r.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get deployment status %s/%s: %w", namespace, name, err)
	}
	var desired int32
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	return map[string]int32{
		"replicas":            desired,
		"ready_replicas":      dep.Status.ReadyReplicas,
		"available_replicas":  dep.Status.AvailableReplicas,
		"updated_replicas":    dep.Status.UpdatedReplicas,
		"unavailable_replicas": dep.Status.UnavailableReplicas,
	}, nil
}

// --- Services ---

// ListServices returns services in the given namespace.
func (r *Resources) ListServices(namespace string) ([]corev1.Service, error) {
	ctx := context.Background()
	svcs, err := r.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services in %s: %w", namespace, err)
	}
	return svcs.Items, nil
}

// GetService returns a single service by namespace and name.
func (r *Resources) GetService(namespace, name string) (*corev1.Service, error) {
	ctx := context.Background()
	svc, err := r.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get service %s/%s: %w", namespace, name, err)
	}
	return svc, nil
}

// GetServiceEndpoints returns the endpoints object for a service.
func (r *Resources) GetServiceEndpoints(namespace, name string) (*corev1.Endpoints, error) {
	ctx := context.Background()
	ep, err := r.clientset.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get endpoints %s/%s: %w", namespace, name, err)
	}
	return ep, nil
}

// --- Events ---

// ListEvents returns events in the given namespace, or all namespaces if empty.
func (r *Resources) ListEvents(namespace string) ([]corev1.Event, error) {
	ctx := context.Background()
	events, err := r.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list events in %s: %w", namespace, err)
	}
	return events.Items, nil
}
