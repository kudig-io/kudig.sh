package audit

import "time"

// AuditEvent represents a single audit log entry.
type AuditEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"eventType"`
	Category  string                 `json:"category"`
	Severity  string                 `json:"severity"`
	Source    string                 `json:"source"`
	User      string                 `json:"user"`
	Action    string                 `json:"action"`
	Resource  map[string]string      `json:"resource"`
	Result    string                 `json:"result"`
	Details   map[string]interface{} `json:"details"`
	IPAddress string                 `json:"ipAddress"`
	UserAgent string                 `json:"userAgent"`
}

// SecurityEvent represents a detected security incident.
type SecurityEvent struct {
	ID                 string              `json:"id"`
	Timestamp          time.Time           `json:"timestamp"`
	Type               string              `json:"type"`
	Severity           string              `json:"severity"`
	Title              string              `json:"title"`
	Description        string              `json:"description"`
	Source             string              `json:"source"`
	AffectedResources  []map[string]string `json:"affectedResources"`
	Remediation        string              `json:"remediation"`
	Status             string              `json:"status"`
	Acknowledged       bool                `json:"acknowledged"`
	AcknowledgedBy     string              `json:"acknowledgedBy,omitempty"`
	AcknowledgedAt     *time.Time          `json:"acknowledgedAt,omitempty"`
	Resolved           bool                `json:"resolved"`
	ResolvedBy         string              `json:"resolvedBy,omitempty"`
	ResolvedAt         *time.Time          `json:"resolvedAt,omitempty"`
}

// ComplianceReport contains the results of a compliance scan.
type ComplianceReport struct {
	ID         string        `json:"id"`
	Timestamp  time.Time     `json:"timestamp"`
	ReportType string        `json:"reportType"`
	Status     string        `json:"status"`
	Summary    ReportSummary `json:"summary"`
	Findings   []Finding     `json:"findings"`
	Score      int           `json:"score"`
}

// ReportSummary provides aggregate counts of findings by severity.
type ReportSummary struct {
	TotalFindings    int `json:"totalFindings"`
	CriticalFindings int `json:"criticalFindings"`
	HighFindings     int `json:"highFindings"`
	MediumFindings   int `json:"mediumFindings"`
	LowFindings      int `json:"lowFindings"`
	InfoFindings     int `json:"infoFindings"`
}

// Finding represents a single compliance finding.
type Finding struct {
	Category       string            `json:"category"`
	Severity       string            `json:"severity"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Resource       map[string]string `json:"resource"`
	Recommendation string            `json:"recommendation"`
}

// AuditFilter is used to filter audit log queries.
type AuditFilter struct {
	EventType string    `json:"eventType,omitempty"`
	Category  string    `json:"category,omitempty"`
	Severity  string    `json:"severity,omitempty"`
	User      string    `json:"user,omitempty"`
	Since     time.Time `json:"since,omitempty"`
	Until     time.Time `json:"until,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// SecurityEventFilter is used to filter security event queries.
type SecurityEventFilter struct {
	Type       string `json:"type,omitempty"`
	Severity   string `json:"severity,omitempty"`
	Status     string `json:"status,omitempty"`
	Unresolved bool   `json:"unresolved,omitempty"`
}
