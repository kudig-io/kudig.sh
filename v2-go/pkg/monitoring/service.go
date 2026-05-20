package monitoring

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Service orchestrates periodic metrics collection and health checking.
type Service struct {
	clientset       kubernetes.Interface
	collector       *Collector
	alerts          map[string]*Alert
	mu              sync.RWMutex
	metricsHistory  []ClusterMetrics
	maxHistorySize  int
	clusterName     string
}

// NewService creates a new monitoring Service for the given cluster.
func NewService(clientset kubernetes.Interface, clusterName string) *Service {
	return &Service{
		clientset:      clientset,
		collector:      NewCollector(clientset),
		alerts:         make(map[string]*Alert),
		metricsHistory: make([]ClusterMetrics, 0),
		maxHistorySize: 120, // Keep ~1 hour of data at 30s intervals
		clusterName:    clusterName,
	}
}

// Start launches two background loops: one for periodic metrics collection
// (every 30 seconds) and one for health/alert checking (every 10 seconds).
// It blocks until ctx is cancelled.
func (s *Service) Start(ctx context.Context) error {
	klog.InfoS("Starting monitoring service", "cluster", s.clusterName)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.metricsCollectionLoop(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.alertCheckingLoop(ctx)
	}()

	wg.Wait()
	return nil
}

// metricsCollectionLoop gathers cluster metrics every 30 seconds.
func (s *Service) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Collect once immediately on startup.
	s.collectAndStore(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectAndStore(ctx)
		}
	}
}

// collectAndStore performs one metrics collection and appends it to history.
func (s *Service) collectAndStore(ctx context.Context) {
	metrics, err := s.collector.CollectClusterMetrics(ctx, s.clusterName)
	if err != nil {
		klog.ErrorS(err, "Failed to collect cluster metrics")
		return
	}

	s.mu.Lock()
	s.metricsHistory = append(s.metricsHistory, *metrics)
	if len(s.metricsHistory) > s.maxHistorySize {
		s.metricsHistory = s.metricsHistory[len(s.metricsHistory)-s.maxHistorySize:]
	}
	s.mu.Unlock()
}

// alertCheckingLoop evaluates cluster health every 10 seconds.
func (s *Service) alertCheckingLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkClusterHealth(ctx)
		}
	}
}

// checkClusterHealth evaluates the latest metrics against predefined thresholds
// and creates alerts when conditions are met.
func (s *Service) checkClusterHealth(ctx context.Context) {
	s.mu.RLock()
	if len(s.metricsHistory) == 0 {
		s.mu.RUnlock()
		return
	}
	latest := s.metricsHistory[len(s.metricsHistory)-1]
	s.mu.RUnlock()

	// Check for not-ready nodes
	if latest.Nodes.NotReady > 0 {
		s.createAlert("warning", "Nodes",
			fmt.Sprintf("%d node(s) in NotReady state", latest.Nodes.NotReady),
			"NodeNotReady",
		)
	}

	// Check for failed pods
	if latest.FailedPods > 0 {
		s.createAlert("error", "Pods",
			fmt.Sprintf("%d pod(s) in Failed state", latest.FailedPods),
			"PodFailed",
		)
	}

	// Check for high pending pod ratio
	if latest.TotalPods > 0 {
		pendingRatio := float64(latest.PendingPods) / float64(latest.TotalPods)
		if pendingRatio > 0.1 {
			s.createAlert("warning", "Pods",
				fmt.Sprintf("%.0f%% pods pending (%d/%d)", pendingRatio*100, latest.PendingPods, latest.TotalPods),
				"HighPendingPods",
			)
		}
	}

	// Check for excessive warning events
	if latest.EventMetrics.Warnings > 100 {
		s.createAlert("warning", "Events",
			fmt.Sprintf("%d warning events in cluster", latest.EventMetrics.Warnings),
			"HighWarningEvents",
		)
	}
}

// createAlert generates a new alert or updates an existing one.
func (s *Service) createAlert(severity, resource, message, reason string) {
	key := fmt.Sprintf("%s:%s:%s", severity, resource, reason)

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.alerts[key]; ok && !existing.Resolved {
		return // Alert already active
	}

	s.alerts[key] = &Alert{
		ID:        generateAlertID(),
		Severity:  severity,
		Resource:  resource,
		Message:   message,
		Reason:    reason,
		Timestamp: time.Now().UTC(),
		Resolved:  false,
	}
}

// GetAlerts returns a copy of all current alerts.
func (s *Service) GetAlerts() []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]Alert, 0, len(s.alerts))
	for _, a := range s.alerts {
		alerts = append(alerts, *a)
	}
	return alerts
}

// ResolveAlert marks the alert with the given ID as resolved.
func (s *Service) ResolveAlert(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, a := range s.alerts {
		if a.ID == id {
			now := time.Now().UTC()
			a.Resolved = true
			a.ResolvedAt = &now
			return true
		}
	}
	return false
}

// GetMetricsHistory returns a copy of the collected metrics history.
func (s *Service) GetMetricsHistory() []ClusterMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := make([]ClusterMetrics, len(s.metricsHistory))
	copy(history, s.metricsHistory)
	return history
}

// generateAlertID creates a short unique identifier for an alert.
func generateAlertID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("alert-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("alert-%x", b)
}
