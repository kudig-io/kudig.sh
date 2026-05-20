package resources

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
)

// Manager wraps a Kubernetes clientset and provides methods to list
// all supported resource types, converting them into ResourceInfo slices.
type Manager struct {
	clientset           *kubernetes.Clientset
	extensionsClientset apiextensionsclientset.Interface
}

// NewManager creates a new resource Manager backed by the given clientset.
func NewManager(clientset *kubernetes.Clientset) *Manager {
	return &Manager{clientset: clientset, extensionsClientset: nil}
}

// NewManagerWithExtensions creates a new resource Manager with full CRD support.
func NewManagerWithExtensions(clientset *kubernetes.Clientset, extClientset apiextensionsclientset.Interface) *Manager {
	return &Manager{clientset: clientset, extensionsClientset: extClientset}
}

// toResourceInfo converts a generic Kubernetes object into a ResourceInfo.
func toResourceInfo(name, namespace, kind string, ts metav1.Time, labels, annotations map[string]string, status string, raw interface{}) ResourceInfo {
	return ResourceInfo{
		Name:             name,
		Namespace:        namespace,
		Kind:             kind,
		CreationTimestamp: ts.Time,
		Labels:           labels,
		Annotations:      annotations,
		Status:           status,
		Raw:              raw,
	}
}

// ---------------------------------------------------------------------------
// CoreV1 resources
// ---------------------------------------------------------------------------

// ListPods returns all pods in the given namespace.
func (m *Manager) ListPods(ns string) ([]ResourceInfo, error) {
	pods, err := m.clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	out := make([]ResourceInfo, 0, len(pods.Items))
	for _, p := range pods.Items {
		status := string(p.Status.Phase)
		out = append(out, toResourceInfo(p.Name, p.Namespace, "Pod", p.CreationTimestamp, p.Labels, p.Annotations, status, p))
	}
	return out, nil
}

// ListServices returns all services in the given namespace.
func (m *Manager) ListServices(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.CoreV1().Services(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, s := range items.Items {
		out = append(out, toResourceInfo(s.Name, s.Namespace, "Service", s.CreationTimestamp, s.Labels, s.Annotations, string(s.Spec.Type), s))
	}
	return out, nil
}

// ListConfigMaps returns all config maps in the given namespace.
func (m *Manager) ListConfigMaps(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.CoreV1().ConfigMaps(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list configmaps: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, cm := range items.Items {
		out = append(out, toResourceInfo(cm.Name, cm.Namespace, "ConfigMap", cm.CreationTimestamp, cm.Labels, cm.Annotations, "", cm))
	}
	return out, nil
}

// ListSecrets returns all secrets in the given namespace.
func (m *Manager) ListSecrets(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.CoreV1().Secrets(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, s := range items.Items {
		out = append(out, toResourceInfo(s.Name, s.Namespace, "Secret", s.CreationTimestamp, s.Labels, s.Annotations, string(s.Type), s))
	}
	return out, nil
}

// ListServiceAccounts returns all service accounts in the given namespace.
func (m *Manager) ListServiceAccounts(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.CoreV1().ServiceAccounts(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list serviceaccounts: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, sa := range items.Items {
		out = append(out, toResourceInfo(sa.Name, sa.Namespace, "ServiceAccount", sa.CreationTimestamp, sa.Labels, sa.Annotations, "", sa))
	}
	return out, nil
}

// ListEndpoints returns all endpoints in the given namespace.
func (m *Manager) ListEndpoints(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.CoreV1().Endpoints(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, ep := range items.Items {
		out = append(out, toResourceInfo(ep.Name, ep.Namespace, "Endpoints", ep.CreationTimestamp, ep.Labels, ep.Annotations, "", ep))
	}
	return out, nil
}

// ListResourceQuotas returns all resource quotas in the given namespace.
func (m *Manager) ListResourceQuotas(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.CoreV1().ResourceQuotas(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list resourcequotas: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, rq := range items.Items {
		out = append(out, toResourceInfo(rq.Name, rq.Namespace, "ResourceQuota", rq.CreationTimestamp, rq.Labels, rq.Annotations, "", rq))
	}
	return out, nil
}

// ListLimitRanges returns all limit ranges in the given namespace.
func (m *Manager) ListLimitRanges(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.CoreV1().LimitRanges(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list limitranges: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, lr := range items.Items {
		out = append(out, toResourceInfo(lr.Name, lr.Namespace, "LimitRange", lr.CreationTimestamp, lr.Labels, lr.Annotations, "", lr))
	}
	return out, nil
}

// ListPersistentVolumes returns all persistent volumes (cluster-scoped).
func (m *Manager) ListPersistentVolumes() ([]ResourceInfo, error) {
	items, err := m.clientset.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list persistentvolumes: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, pv := range items.Items {
		status := string(pv.Status.Phase)
		out = append(out, toResourceInfo(pv.Name, "", "PersistentVolume", pv.CreationTimestamp, pv.Labels, pv.Annotations, status, pv))
	}
	return out, nil
}

// ListPersistentVolumeClaims returns all PVCs in the given namespace.
func (m *Manager) ListPersistentVolumeClaims(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list persistentvolumeclaims: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, pvc := range items.Items {
		status := string(pvc.Status.Phase)
		out = append(out, toResourceInfo(pvc.Name, pvc.Namespace, "PersistentVolumeClaim", pvc.CreationTimestamp, pvc.Labels, pvc.Annotations, status, pvc))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// AppsV1 resources
// ---------------------------------------------------------------------------

// ListDeployments returns all deployments in the given namespace.
func (m *Manager) ListDeployments(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, d := range items.Items {
		status := fmt.Sprintf("ready=%d/%d", d.Status.ReadyReplicas, d.Status.Replicas)
		out = append(out, toResourceInfo(d.Name, d.Namespace, "Deployment", d.CreationTimestamp, d.Labels, d.Annotations, status, d))
	}
	return out, nil
}

// ListStatefulSets returns all stateful sets in the given namespace.
func (m *Manager) ListStatefulSets(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.AppsV1().StatefulSets(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list statefulsets: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, s := range items.Items {
		status := fmt.Sprintf("ready=%d/%d", s.Status.ReadyReplicas, s.Status.Replicas)
		out = append(out, toResourceInfo(s.Name, s.Namespace, "StatefulSet", s.CreationTimestamp, s.Labels, s.Annotations, status, s))
	}
	return out, nil
}

// ListDaemonSets returns all daemon sets in the given namespace.
func (m *Manager) ListDaemonSets(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.AppsV1().DaemonSets(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list daemonsets: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, ds := range items.Items {
		status := fmt.Sprintf("ready=%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
		out = append(out, toResourceInfo(ds.Name, ds.Namespace, "DaemonSet", ds.CreationTimestamp, ds.Labels, ds.Annotations, status, ds))
	}
	return out, nil
}

// ListReplicaSets returns all replica sets in the given namespace.
func (m *Manager) ListReplicaSets(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list replicasets: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, rs := range items.Items {
		status := fmt.Sprintf("ready=%d/%d", rs.Status.ReadyReplicas, rs.Status.Replicas)
		out = append(out, toResourceInfo(rs.Name, rs.Namespace, "ReplicaSet", rs.CreationTimestamp, rs.Labels, rs.Annotations, status, rs))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// BatchV1 resources
// ---------------------------------------------------------------------------

// ListJobs returns all jobs in the given namespace.
func (m *Manager) ListJobs(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.BatchV1().Jobs(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, j := range items.Items {
		status := ""
		if c := len(j.Status.Conditions); c > 0 {
			status = string(j.Status.Conditions[c-1].Type)
		}
		out = append(out, toResourceInfo(j.Name, j.Namespace, "Job", j.CreationTimestamp, j.Labels, j.Annotations, status, j))
	}
	return out, nil
}

// ListCronJobs returns all cron jobs in the given namespace.
func (m *Manager) ListCronJobs(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.BatchV1().CronJobs(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list cronjobs: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, cj := range items.Items {
		out = append(out, toResourceInfo(cj.Name, cj.Namespace, "CronJob", cj.CreationTimestamp, cj.Labels, cj.Annotations, "", cj))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// NetworkingV1 resources
// ---------------------------------------------------------------------------

// ListIngresses returns all ingresses in the given namespace.
func (m *Manager) ListIngresses(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.NetworkingV1().Ingresses(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list ingresses: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, ing := range items.Items {
		out = append(out, toResourceInfo(ing.Name, ing.Namespace, "Ingress", ing.CreationTimestamp, ing.Labels, ing.Annotations, "", ing))
	}
	return out, nil
}

// ListNetworkPolicies returns all network policies in the given namespace.
func (m *Manager) ListNetworkPolicies(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.NetworkingV1().NetworkPolicies(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list networkpolicies: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, np := range items.Items {
		out = append(out, toResourceInfo(np.Name, np.Namespace, "NetworkPolicy", np.CreationTimestamp, np.Labels, np.Annotations, "", np))
	}
	return out, nil
}

// ListIngressClasses returns all ingress classes (cluster-scoped).
func (m *Manager) ListIngressClasses() ([]ResourceInfo, error) {
	items, err := m.clientset.NetworkingV1().IngressClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list ingressclasses: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, ic := range items.Items {
		out = append(out, toResourceInfo(ic.Name, "", "IngressClass", ic.CreationTimestamp, ic.Labels, ic.Annotations, "", ic))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// RBAC resources
// ---------------------------------------------------------------------------

// ListRoles returns all roles in the given namespace.
func (m *Manager) ListRoles(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.RbacV1().Roles(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, r := range items.Items {
		out = append(out, toResourceInfo(r.Name, r.Namespace, "Role", r.CreationTimestamp, r.Labels, r.Annotations, "", r))
	}
	return out, nil
}

// ListClusterRoles returns all cluster roles (cluster-scoped).
func (m *Manager) ListClusterRoles() ([]ResourceInfo, error) {
	items, err := m.clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list clusterroles: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, cr := range items.Items {
		out = append(out, toResourceInfo(cr.Name, "", "ClusterRole", cr.CreationTimestamp, cr.Labels, cr.Annotations, "", cr))
	}
	return out, nil
}

// ListRoleBindings returns all role bindings in the given namespace.
func (m *Manager) ListRoleBindings(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.RbacV1().RoleBindings(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list rolebindings: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, rb := range items.Items {
		out = append(out, toResourceInfo(rb.Name, rb.Namespace, "RoleBinding", rb.CreationTimestamp, rb.Labels, rb.Annotations, "", rb))
	}
	return out, nil
}

// ListClusterRoleBindings returns all cluster role bindings (cluster-scoped).
func (m *Manager) ListClusterRoleBindings() ([]ResourceInfo, error) {
	items, err := m.clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list clusterrolebindings: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, crb := range items.Items {
		out = append(out, toResourceInfo(crb.Name, "", "ClusterRoleBinding", crb.CreationTimestamp, crb.Labels, crb.Annotations, "", crb))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// StorageV1 resources
// ---------------------------------------------------------------------------

// ListStorageClasses returns all storage classes (cluster-scoped).
func (m *Manager) ListStorageClasses() ([]ResourceInfo, error) {
	items, err := m.clientset.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list storageclasses: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, sc := range items.Items {
		out = append(out, toResourceInfo(sc.Name, "", "StorageClass", sc.CreationTimestamp, sc.Labels, sc.Annotations, "", sc))
	}
	return out, nil
}

// ListCSIDrivers returns all CSI drivers (cluster-scoped).
func (m *Manager) ListCSIDrivers() ([]ResourceInfo, error) {
	items, err := m.clientset.StorageV1().CSIDrivers().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list csidrivers: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, d := range items.Items {
		out = append(out, toResourceInfo(d.Name, "", "CSIDriver", d.CreationTimestamp, d.Labels, d.Annotations, "", d))
	}
	return out, nil
}

// ListCSINodes returns all CSI nodes (cluster-scoped).
func (m *Manager) ListCSINodes() ([]ResourceInfo, error) {
	items, err := m.clientset.StorageV1().CSINodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list csinodes: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, n := range items.Items {
		out = append(out, toResourceInfo(n.Name, "", "CSINode", n.CreationTimestamp, n.Labels, n.Annotations, "", n))
	}
	return out, nil
}

// ListVolumeAttachments returns all volume attachments (cluster-scoped).
func (m *Manager) ListVolumeAttachments() ([]ResourceInfo, error) {
	items, err := m.clientset.StorageV1().VolumeAttachments().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list volumeattachments: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, va := range items.Items {
		status := ""
		if va.Status.Attached {
			status = "Attached"
		}
		out = append(out, toResourceInfo(va.Name, "", "VolumeAttachment", va.CreationTimestamp, va.Labels, va.Annotations, status, va))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// AutoscalingV2 resources
// ---------------------------------------------------------------------------

// ListHPAs returns all horizontal pod autoscalers in the given namespace.
func (m *Manager) ListHPAs(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.AutoscalingV2().HorizontalPodAutoscalers(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list hpas: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, hpa := range items.Items {
		status := fmt.Sprintf("ready=%d", hpa.Status.CurrentReplicas)
		out = append(out, toResourceInfo(hpa.Name, hpa.Namespace, "HorizontalPodAutoscaler", hpa.CreationTimestamp, hpa.Labels, hpa.Annotations, status, hpa))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// PolicyV1 resources
// ---------------------------------------------------------------------------

// ListPDBs returns all pod disruption budgets in the given namespace.
func (m *Manager) ListPDBs(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.PolicyV1().PodDisruptionBudgets(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pdbs: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, pdb := range items.Items {
		status := fmt.Sprintf("available=%d", pdb.Status.CurrentHealthy)
		out = append(out, toResourceInfo(pdb.Name, pdb.Namespace, "PodDisruptionBudget", pdb.CreationTimestamp, pdb.Labels, pdb.Annotations, status, pdb))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// SchedulingV1 resources
// ---------------------------------------------------------------------------

// ListPriorityClasses returns all priority classes (cluster-scoped).
func (m *Manager) ListPriorityClasses() ([]ResourceInfo, error) {
	items, err := m.clientset.SchedulingV1().PriorityClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list priorityclasses: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, pc := range items.Items {
		out = append(out, toResourceInfo(pc.Name, "", "PriorityClass", pc.CreationTimestamp, pc.Labels, pc.Annotations, "", pc))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// CoordinationV1 resources
// ---------------------------------------------------------------------------

// ListLeases returns all leases in the given namespace.
func (m *Manager) ListLeases(ns string) ([]ResourceInfo, error) {
	items, err := m.clientset.CoordinationV1().Leases(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list leases: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, l := range items.Items {
		holder := ""
		if l.Spec.HolderIdentity != nil {
			holder = *l.Spec.HolderIdentity
		}
		out = append(out, toResourceInfo(l.Name, l.Namespace, "Lease", l.CreationTimestamp, l.Labels, l.Annotations, holder, l))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// NodeV1 resources
// ---------------------------------------------------------------------------

// ListRuntimeClasses returns all runtime classes (cluster-scoped).
func (m *Manager) ListRuntimeClasses() ([]ResourceInfo, error) {
	items, err := m.clientset.NodeV1().RuntimeClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list runtimeclasses: %w", err)
	}
	out := make([]ResourceInfo, 0, len(items.Items))
	for _, rc := range items.Items {
		out = append(out, toResourceInfo(rc.Name, "", "RuntimeClass", rc.CreationTimestamp, rc.Labels, rc.Annotations, "", rc))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// APIExtensionsV1 resources
// ---------------------------------------------------------------------------

// ListCRDs returns all custom resource definitions (cluster-scoped).
func (m *Manager) ListCRDs() ([]ResourceInfo, error) {
	if m.extensionsClientset == nil {
		return nil, fmt.Errorf("apiextensions clientset not configured; use NewManagerWithExtensions")
	}
	crds, err := m.extensionsClientset.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list crds: %w", err)
	}
	out := make([]ResourceInfo, 0, len(crds.Items))
	for _, crd := range crds.Items {
		status := ""
		if len(crd.Status.Conditions) > 0 {
			status = string(crd.Status.Conditions[len(crd.Status.Conditions)-1].Status)
		}
		out = append(out, toResourceInfo(crd.Name, "", "CustomResourceDefinition", crd.CreationTimestamp, crd.Labels, crd.Annotations, status, crd))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Aggregate helper
// ---------------------------------------------------------------------------

// GetResourceCounts returns a count of each resource type across all
// supported API groups. Cluster-scoped resources are listed without a
// namespace; namespace-scoped resources use the "default" namespace.
func GetResourceCounts(clientset *kubernetes.Clientset) map[string]int {
	return GetResourceCountsWithExtensions(clientset, nil)
}

// GetResourceCountsWithExtensions returns counts including CRDs when an apiextensions clientset is provided.
func GetResourceCountsWithExtensions(clientset *kubernetes.Clientset, extClientset apiextensionsclientset.Interface) map[string]int {
	counts := make(map[string]int)
	ctx := context.TODO()
	opts := metav1.ListOptions{}

	type listFunc struct {
		name string
		fn   func() (int, error)
	}

	listers := []listFunc{
		{"Pods", func() (int, error) {
			l, e := clientset.CoreV1().Pods("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"Services", func() (int, error) {
			l, e := clientset.CoreV1().Services("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"ConfigMaps", func() (int, error) {
			l, e := clientset.CoreV1().ConfigMaps("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"Secrets", func() (int, error) {
			l, e := clientset.CoreV1().Secrets("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"ServiceAccounts", func() (int, error) {
			l, e := clientset.CoreV1().ServiceAccounts("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"Endpoints", func() (int, error) {
			l, e := clientset.CoreV1().Endpoints("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"ResourceQuotas", func() (int, error) {
			l, e := clientset.CoreV1().ResourceQuotas("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"LimitRanges", func() (int, error) {
			l, e := clientset.CoreV1().LimitRanges("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"PersistentVolumes", func() (int, error) {
			l, e := clientset.CoreV1().PersistentVolumes().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"PersistentVolumeClaims", func() (int, error) {
			l, e := clientset.CoreV1().PersistentVolumeClaims("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"Deployments", func() (int, error) {
			l, e := clientset.AppsV1().Deployments("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"StatefulSets", func() (int, error) {
			l, e := clientset.AppsV1().StatefulSets("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"DaemonSets", func() (int, error) {
			l, e := clientset.AppsV1().DaemonSets("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"ReplicaSets", func() (int, error) {
			l, e := clientset.AppsV1().ReplicaSets("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"Jobs", func() (int, error) {
			l, e := clientset.BatchV1().Jobs("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"CronJobs", func() (int, error) {
			l, e := clientset.BatchV1().CronJobs("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"Ingresses", func() (int, error) {
			l, e := clientset.NetworkingV1().Ingresses("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"NetworkPolicies", func() (int, error) {
			l, e := clientset.NetworkingV1().NetworkPolicies("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"IngressClasses", func() (int, error) {
			l, e := clientset.NetworkingV1().IngressClasses().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"Roles", func() (int, error) {
			l, e := clientset.RbacV1().Roles("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"ClusterRoles", func() (int, error) {
			l, e := clientset.RbacV1().ClusterRoles().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"RoleBindings", func() (int, error) {
			l, e := clientset.RbacV1().RoleBindings("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"ClusterRoleBindings", func() (int, error) {
			l, e := clientset.RbacV1().ClusterRoleBindings().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"StorageClasses", func() (int, error) {
			l, e := clientset.StorageV1().StorageClasses().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"CSIDrivers", func() (int, error) {
			l, e := clientset.StorageV1().CSIDrivers().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"CSINodes", func() (int, error) {
			l, e := clientset.StorageV1().CSINodes().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"VolumeAttachments", func() (int, error) {
			l, e := clientset.StorageV1().VolumeAttachments().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"HorizontalPodAutoscalers", func() (int, error) {
			l, e := clientset.AutoscalingV2().HorizontalPodAutoscalers("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"PodDisruptionBudgets", func() (int, error) {
			l, e := clientset.PolicyV1().PodDisruptionBudgets("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"PriorityClasses", func() (int, error) {
			l, e := clientset.SchedulingV1().PriorityClasses().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"Leases", func() (int, error) {
			l, e := clientset.CoordinationV1().Leases("").List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"RuntimeClasses", func() (int, error) {
			l, e := clientset.NodeV1().RuntimeClasses().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
		{"CustomResourceDefinitions", func() (int, error) {
			if extClientset == nil {
				return 0, fmt.Errorf("apiextensions clientset not configured")
			}
			l, e := extClientset.ApiextensionsV1().CustomResourceDefinitions().List(ctx, opts)
			if e != nil { return 0, e }
			return len(l.Items), nil
		}},
	}

	for _, lf := range listers {
		n, err := lf.fn()
		if err == nil {
			counts[lf.name] = n
		}
	}
	return counts
}
