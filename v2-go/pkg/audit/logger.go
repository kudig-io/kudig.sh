package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Logger manages audit logs, security events, and compliance reports.
type Logger struct {
	auditLogs        []AuditEvent
	securityEvents   []SecurityEvent
	complianceReports []ComplianceReport
	mu               sync.RWMutex
	dataDir          string
}

// NewLogger creates a new audit logger.
func NewLogger(dataDir string) *Logger {
	return &Logger{
		dataDir: dataDir,
	}
}

// LogEvent records an audit event and persists it.
func (l *Logger) LogEvent(event AuditEvent) AuditEvent {
	l.mu.Lock()
	defer l.mu.Unlock()

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	l.auditLogs = append(l.auditLogs, event)
	l.saveAuditLogsLocked()
	return event
}

// GetLogs returns audit logs matching the filter.
func (l *Logger) GetLogs(filter AuditFilter) []AuditEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []AuditEvent
	for i := len(l.auditLogs) - 1; i >= 0; i-- {
		e := l.auditLogs[i]
		if filter.EventType != "" && e.EventType != filter.EventType {
			continue
		}
		if filter.Category != "" && e.Category != filter.Category {
			continue
		}
		if filter.Severity != "" && e.Severity != filter.Severity {
			continue
		}
		if filter.User != "" && e.User != filter.User {
			continue
		}
		if !filter.Since.IsZero() && e.Timestamp.Before(filter.Since) {
			continue
		}
		if !filter.Until.IsZero() && e.Timestamp.After(filter.Until) {
			continue
		}
		result = append(result, e)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result
}

// GetLog returns a single audit event by ID.
func (l *Logger) GetLog(id string) AuditEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, e := range l.auditLogs {
		if e.ID == id {
			return e
		}
	}
	return AuditEvent{}
}

// CreateSecurityEvent records a new security event.
func (l *Logger) CreateSecurityEvent(event SecurityEvent) SecurityEvent {
	l.mu.Lock()
	defer l.mu.Unlock()

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Status == "" {
		event.Status = "open"
	}

	l.securityEvents = append(l.securityEvents, event)
	l.saveSecurityEventsLocked()
	return event
}

// GetSecurityEvents returns security events matching the filter.
func (l *Logger) GetSecurityEvents(filter SecurityEventFilter) []SecurityEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []SecurityEvent
	for _, e := range l.securityEvents {
		if filter.Type != "" && e.Type != filter.Type {
			continue
		}
		if filter.Severity != "" && e.Severity != filter.Severity {
			continue
		}
		if filter.Status != "" && e.Status != filter.Status {
			continue
		}
		if filter.Unresolved && e.Resolved {
			continue
		}
		result = append(result, e)
	}
	return result
}

// GetSecurityEvent returns a single security event by ID.
func (l *Logger) GetSecurityEvent(id string) SecurityEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, e := range l.securityEvents {
		if e.ID == id {
			return e
		}
	}
	return SecurityEvent{}
}

// AcknowledgeSecurityEvent marks a security event as acknowledged.
func (l *Logger) AcknowledgeSecurityEvent(id, by string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i, e := range l.securityEvents {
		if e.ID == id {
			now := time.Now().UTC()
			l.securityEvents[i].Acknowledged = true
			l.securityEvents[i].AcknowledgedBy = by
			l.securityEvents[i].AcknowledgedAt = &now
			l.saveSecurityEventsLocked()
			return nil
		}
	}
	return fmt.Errorf("security event %s not found", id)
}

// ResolveSecurityEvent marks a security event as resolved.
func (l *Logger) ResolveSecurityEvent(id, by string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i, e := range l.securityEvents {
		if e.ID == id {
			now := time.Now().UTC()
			l.securityEvents[i].Resolved = true
			l.securityEvents[i].ResolvedBy = by
			l.securityEvents[i].ResolvedAt = &now
			l.securityEvents[i].Status = "resolved"
			l.saveSecurityEventsLocked()
			return nil
		}
	}
	return fmt.Errorf("security event %s not found", id)
}

// GenerateComplianceReport runs a compliance scan against the cluster.
func (l *Logger) GenerateComplianceReport(ctx context.Context, clientset kubernetes.Interface, reportType string) (*ComplianceReport, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var findings []Finding

	podFindings, err := l.checkPodSecurity(ctx, clientset)
	if err != nil {
		return nil, fmt.Errorf("check pod security: %w", err)
	}
	findings = append(findings, podFindings...)

	nodeFindings, err := l.checkNodeSecurity(ctx, clientset)
	if err != nil {
		return nil, fmt.Errorf("check node security: %w", err)
	}
	findings = append(findings, nodeFindings...)

	nsFindings, err := l.checkNamespaceSecurity(ctx, clientset)
	if err != nil {
		return nil, fmt.Errorf("check namespace security: %w", err)
	}
	findings = append(findings, nsFindings...)

	secretFindings, err := l.checkSecretSecurity(ctx, clientset)
	if err != nil {
		return nil, fmt.Errorf("check secret security: %w", err)
	}
	findings = append(findings, secretFindings...)

	rbacFindings, err := l.checkRBACSecurity(ctx, clientset)
	if err != nil {
		return nil, fmt.Errorf("check rbac security: %w", err)
	}
	findings = append(findings, rbacFindings...)

	networkFindings, err := l.checkNetworkSecurity(ctx, clientset)
	if err != nil {
		return nil, fmt.Errorf("check network security: %w", err)
	}
	findings = append(findings, networkFindings...)

	summary := l.buildSummary(findings)
	score := l.calculateComplianceScore(findings)

	report := ComplianceReport{
		ID:         uuid.New().String(),
		Timestamp:  time.Now().UTC(),
		ReportType: reportType,
		Status:     "completed",
		Summary:    summary,
		Findings:   findings,
		Score:      score,
	}

	l.complianceReports = append(l.complianceReports, report)
	l.saveComplianceReportsLocked()
	return &report, nil
}

func (l *Logger) checkPodSecurity(ctx context.Context, clientset kubernetes.Interface) ([]Finding, error) {
	var findings []Finding

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pod := range pods.Items {
		res := map[string]string{"kind": "pod", "namespace": pod.Namespace, "name": pod.Name}

		for _, c := range pod.Spec.Containers {
			if c.SecurityContext != nil && c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
				findings = append(findings, Finding{
					Category:       "pod-security",
					Severity:       "critical",
					Title:          "Privileged container detected",
					Description:    fmt.Sprintf("Container %s in pod %s/%s runs in privileged mode", c.Name, pod.Namespace, pod.Name),
					Resource:       res,
					Recommendation: "Remove privileged flag and use specific capabilities instead",
				})
			}

			if c.SecurityContext == nil {
				findings = append(findings, Finding{
					Category:       "pod-security",
					Severity:       "medium",
					Title:          "No securityContext defined",
					Description:    fmt.Sprintf("Container %s in pod %s/%s has no securityContext", c.Name, pod.Namespace, pod.Name),
					Resource:       res,
					Recommendation: "Define a securityContext with runAsNonRoot, readOnlyRootFilesystem, and drop all capabilities",
				})
			}

			if len(c.Resources.Limits) == 0 && len(c.Resources.Requests) == 0 {
				findings = append(findings, Finding{
					Category:       "pod-security",
					Severity:       "medium",
					Title:          "No resource limits defined",
					Description:    fmt.Sprintf("Container %s in pod %s/%s has no resource limits or requests", c.Name, pod.Namespace, pod.Name),
					Resource:       res,
					Recommendation: "Set CPU and memory requests and limits to prevent resource exhaustion",
				})
			}
		}
	}

	return findings, nil
}

func (l *Logger) checkNodeSecurity(ctx context.Context, clientset kubernetes.Interface) ([]Finding, error) {
	var findings []Finding

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodes.Items {
		res := map[string]string{"kind": "node", "name": node.Name}

		if len(node.Spec.Taints) == 0 {
			findings = append(findings, Finding{
				Category:       "node-security",
				Severity:       "low",
				Title:          "Node has no taints",
				Description:    fmt.Sprintf("Node %s has no taints, allowing any pod to be scheduled", node.Name),
				Resource:       res,
				Recommendation: "Apply appropriate taints to restrict workload placement",
			})
		}

		hasRole := false
		for label := range node.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {
				hasRole = true
				break
			}
		}
		if !hasRole {
			findings = append(findings, Finding{
				Category:       "node-security",
				Severity:       "info",
				Title:          "No node role labels",
				Description:    fmt.Sprintf("Node %s has no role labels", node.Name),
				Resource:       res,
				Recommendation: "Assign node role labels for better scheduling control",
			})
		}
	}

	return findings, nil
}

func (l *Logger) checkNamespaceSecurity(ctx context.Context, clientset kubernetes.Interface) ([]Finding, error) {
	var findings []Finding

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, ns := range namespaces.Items {
		res := map[string]string{"kind": "namespace", "name": ns.Name}

		if ns.Name == "default" {
			findings = append(findings, Finding{
				Category:       "namespace-security",
				Severity:       "medium",
				Title:          "Workloads in default namespace",
				Description:    "The default namespace should not be used for production workloads",
				Resource:       res,
				Recommendation: "Move workloads to dedicated namespaces",
			})
		}

		if len(ns.Labels) <= 1 { // only kubernetes.io/metadata.name
			findings = append(findings, Finding{
				Category:       "namespace-security",
				Severity:       "low",
				Title:          "Namespace has minimal labels",
				Description:    fmt.Sprintf("Namespace %s has no organizational labels", ns.Name),
				Resource:       res,
				Recommendation: "Add labels for environment, team, and compliance tracking",
			})
		}
	}

	return findings, nil
}

func (l *Logger) checkSecretSecurity(ctx context.Context, clientset kubernetes.Interface) ([]Finding, error) {
	var findings []Finding

	secrets, err := clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets.Items {
		res := map[string]string{"kind": "secret", "namespace": secret.Namespace, "name": secret.Name}

		if secret.Type != corev1.SecretTypeOpaque {
			continue // only check opaque secrets
		}

		if len(secret.Annotations) == 0 {
			findings = append(findings, Finding{
				Category:       "secret-security",
				Severity:       "low",
				Title:          "Secret without annotations",
				Description:    fmt.Sprintf("Secret %s/%s has no annotations for tracking or rotation", secret.Namespace, secret.Name),
				Resource:       res,
				Recommendation: "Add annotations for creation date, owner, and rotation schedule",
			})
		}
	}

	return findings, nil
}

func (l *Logger) checkRBACSecurity(ctx context.Context, clientset kubernetes.Interface) ([]Finding, error) {
	var findings []Finding

	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, crb := range clusterRoleBindings.Items {
		if crb.RoleRef.Name == "cluster-admin" {
			for _, subject := range crb.Subjects {
				if subject.Kind == "User" || subject.Kind == "ServiceAccount" {
					findings = append(findings, Finding{
						Category: "rbac-security",
						Severity: "high",
						Title:    "Cluster-admin binding detected",
						Description: fmt.Sprintf("Subject %s/%s bound to cluster-admin via %s",
							subject.Kind, subject.Name, crb.Name),
						Resource:       map[string]string{"kind": "clusterrolebinding", "name": crb.Name},
						Recommendation: "Use more restrictive roles with specific permissions",
					})
				}
			}
		}
	}

	return findings, nil
}

func (l *Logger) checkNetworkSecurity(ctx context.Context, clientset kubernetes.Interface) ([]Finding, error) {
	var findings []Finding

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, ns := range namespaces.Items {
		if ns.Name == "kube-system" || ns.Name == "kube-public" || ns.Name == "kube-node-lease" {
			continue
		}

		policies, err := clientset.NetworkingV1().NetworkPolicies(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		hasDefaultDeny := false
		for _, np := range policies.Items {
			for _, pt := range np.Spec.PolicyTypes {
				if pt == networkingv1.PolicyTypeIngress && len(np.Spec.Ingress) == 0 &&
					len(np.Spec.PodSelector.MatchLabels) == 0 && np.Spec.PodSelector.MatchExpressions == nil {
					hasDefaultDeny = true
				}
			}
		}

		if !hasDefaultDeny && len(policies.Items) == 0 {
			findings = append(findings, Finding{
				Category:       "network-security",
				Severity:       "medium",
				Title:          "No network policies",
				Description:    fmt.Sprintf("Namespace %s has no network policies defined", ns.Name),
				Resource:       map[string]string{"kind": "namespace", "name": ns.Name},
				Recommendation: "Implement default-deny network policies and whitelist required traffic",
			})
		}
	}

	return findings, nil
}

func (l *Logger) buildSummary(findings []Finding) ReportSummary {
	var summary ReportSummary
	summary.TotalFindings = len(findings)
	for _, f := range findings {
		switch strings.ToLower(f.Severity) {
		case "critical":
			summary.CriticalFindings++
		case "high":
			summary.HighFindings++
		case "medium":
			summary.MediumFindings++
		case "low":
			summary.LowFindings++
		case "info":
			summary.InfoFindings++
		}
	}
	return summary
}

// calculateComplianceScore computes a 0-100 score based on findings severity.
func (l *Logger) calculateComplianceScore(findings []Finding) int {
	score := 100
	for _, f := range findings {
		switch strings.ToLower(f.Severity) {
		case "critical":
			score -= 10
		case "high":
			score -= 5
		case "medium":
			score -= 3
		case "low":
			score -= 1
		case "info":
			score -= 0
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

// Persistence methods

func (l *Logger) saveAuditLogsLocked() {
	if l.dataDir == "" {
		return
	}
	os.MkdirAll(l.dataDir, 0o755)
	data, _ := json.MarshalIndent(l.auditLogs, "", "  ")
	os.WriteFile(filepath.Join(l.dataDir, "audit_logs.json"), data, 0o644)
}

func (l *Logger) saveSecurityEventsLocked() {
	if l.dataDir == "" {
		return
	}
	os.MkdirAll(l.dataDir, 0o755)
	data, _ := json.MarshalIndent(l.securityEvents, "", "  ")
	os.WriteFile(filepath.Join(l.dataDir, "security_events.json"), data, 0o644)
}

func (l *Logger) saveComplianceReportsLocked() {
	if l.dataDir == "" {
		return
	}
	os.MkdirAll(l.dataDir, 0o755)
	data, _ := json.MarshalIndent(l.complianceReports, "", "  ")
	os.WriteFile(filepath.Join(l.dataDir, "compliance_reports.json"), data, 0o644)
}

// LoadAuditData reads all audit data from disk.
func (l *Logger) LoadAuditData() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(l.dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	if data, err := os.ReadFile(filepath.Join(l.dataDir, "audit_logs.json")); err == nil {
		var logs []AuditEvent
		if err := json.Unmarshal(data, &logs); err == nil {
			l.auditLogs = logs
		}
	}

	if data, err := os.ReadFile(filepath.Join(l.dataDir, "security_events.json")); err == nil {
		var events []SecurityEvent
		if err := json.Unmarshal(data, &events); err == nil {
			l.securityEvents = events
		}
	}

	if data, err := os.ReadFile(filepath.Join(l.dataDir, "compliance_reports.json")); err == nil {
		var reports []ComplianceReport
		if err := json.Unmarshal(data, &reports); err == nil {
			l.complianceReports = reports
		}
	}

	return nil
}

// SaveAuditData persists all audit data to disk.
func (l *Logger) SaveAuditData() error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if err := os.MkdirAll(l.dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	l.saveAuditLogsLocked()
	l.saveSecurityEventsLocked()
	l.saveComplianceReportsLocked()
	return nil
}
