// Package storageanalysis provides Kubernetes storage resource analysis.
package storageanalysis

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Analyzer performs storage analysis against a Kubernetes cluster.
type Analyzer struct {
	clientset *kubernetes.Clientset
}

// NewAnalyzer creates a new storage Analyzer.
func NewAnalyzer(clientset *kubernetes.Clientset) *Analyzer {
	return &Analyzer{clientset: clientset}
}

// CapacityInfo holds aggregate storage capacity numbers.
type CapacityInfo struct {
	TotalBytes     int64 `json:"total_bytes"`
	UsedBytes      int64 `json:"used_bytes"`
	AvailableBytes int64 `json:"available_bytes"`
}

// StorageAnalysis holds the result of a full storage analysis.
type StorageAnalysis struct {
	TotalPVs           int                 `json:"total_pvs"`
	TotalPVCs          int                 `json:"total_pvcs"`
	TotalStorageClasses int               `json:"total_storage_classes"`
	PVByStatus         map[string][]string `json:"pv_by_status"`
	PVCByStatus        map[string][]string `json:"pvc_by_status"`
	PVByStorageClass   map[string][]string `json:"pv_by_storage_class"`
	StorageCapacity    CapacityInfo        `json:"storage_capacity"`
	SCByProvisioner    map[string][]string `json:"sc_by_provisioner"`
	Timestamp          time.Time           `json:"timestamp"`
}

// ListPersistentVolumes returns all PersistentVolumes in the cluster.
func (a *Analyzer) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	list, err := a.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListStorageClasses returns all StorageClasses in the cluster.
func (a *Analyzer) ListStorageClasses(ctx context.Context) ([]storagev1.StorageClass, error) {
	list, err := a.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// AnalyzeStorage collects PVs, PVCs, and StorageClasses and produces a
// StorageAnalysis.
func (a *Analyzer) AnalyzeStorage(ctx context.Context) (*StorageAnalysis, error) {
	pvs, err := a.ListPersistentVolumes(ctx)
	if err != nil {
		return nil, err
	}

	pvcs, err := a.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	scs, err := a.ListStorageClasses(ctx)
	if err != nil {
		return nil, err
	}

	analysis := &StorageAnalysis{
		TotalPVs:            len(pvs),
		TotalPVCs:           len(pvcs.Items),
		TotalStorageClasses: len(scs),
		PVByStatus:          make(map[string][]string),
		PVCByStatus:         make(map[string][]string),
		PVByStorageClass:    make(map[string][]string),
		SCByProvisioner:     make(map[string][]string),
		Timestamp:           time.Now(),
	}

	// Index PVs by status and storage class, accumulate capacity.
	for _, pv := range pvs {
		status := string(pv.Status.Phase)
		analysis.PVByStatus[status] = append(analysis.PVByStatus[status], pv.Name)

		sc := pv.Spec.StorageClassName
		if sc == "" {
			sc = "<none>"
		}
		analysis.PVByStorageClass[sc] = append(analysis.PVByStorageClass[sc], pv.Name)

		if cap, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
			analysis.StorageCapacity.TotalBytes += cap.Value()
		}
	}

	// Index PVCs by status.
	for _, pvc := range pvcs.Items {
		status := string(pvc.Status.Phase)
		analysis.PVCByStatus[status] = append(analysis.PVCByStatus[status], pvc.Name)
	}

	// Index StorageClasses by provisioner.
	for _, sc := range scs {
		provisioner := sc.Provisioner
		analysis.SCByProvisioner[provisioner] = append(analysis.SCByProvisioner[provisioner], sc.Name)
	}

	// Derive available bytes.
	analysis.StorageCapacity.AvailableBytes = analysis.StorageCapacity.TotalBytes - analysis.StorageCapacity.UsedBytes
	if analysis.StorageCapacity.AvailableBytes < 0 {
		analysis.StorageCapacity.AvailableBytes = 0
	}

	return analysis, nil
}
