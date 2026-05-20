package alerting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Engine manages alert rules, evaluates conditions against data, and tracks alert history.
type Engine struct {
	rules       []AlertRule
	history     []AlertRecord
	activeAlerts map[string]*AlertRecord
	mu          sync.RWMutex
	rulesFile   string
	historyFile string
}

// NewEngine creates a new alerting engine with the given file paths for persistence.
func NewEngine(rulesFile, historyFile string) *Engine {
	return &Engine{
		activeAlerts: make(map[string]*AlertRecord),
		rulesFile:    rulesFile,
		historyFile:  historyFile,
	}
}

// LoadRules loads alert rules from the configured file, falling back to defaults on error.
func (e *Engine) LoadRules() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, err := os.ReadFile(e.rulesFile)
	if err != nil {
		e.rules = e.getDefaultRules()
		return nil
	}

	var rules []AlertRule
	if err := json.Unmarshal(data, &rules); err != nil {
		e.rules = e.getDefaultRules()
		return nil
	}

	if len(rules) == 0 {
		e.rules = e.getDefaultRules()
		return nil
	}

	e.rules = rules
	return nil
}

// SaveRules persists the current rules to the configured file.
func (e *Engine) SaveRules() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(e.rulesFile), 0755); err != nil {
		return fmt.Errorf("creating rules directory: %w", err)
	}

	data, err := json.MarshalIndent(e.rules, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling rules: %w", err)
	}

	if err := os.WriteFile(e.rulesFile, data, 0644); err != nil {
		return fmt.Errorf("writing rules file: %w", err)
	}

	return nil
}

// AddRule adds a new alert rule with an auto-generated ID and timestamps.
func (e *Engine) AddRule(rule AlertRule) (*AlertRule, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rule.ID = e.generateID()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = rule.CreatedAt

	e.rules = append(e.rules, rule)
	return &rule, nil
}

// UpdateRule updates an existing rule identified by id with the provided fields.
func (e *Engine) UpdateRule(id string, updates AlertRule) (*AlertRule, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.rules {
		if e.rules[i].ID == id {
			if updates.Name != "" {
				e.rules[i].Name = updates.Name
			}
			if updates.Description != "" {
				e.rules[i].Description = updates.Description
			}
			e.rules[i].Enabled = updates.Enabled
			if updates.Severity != "" {
				e.rules[i].Severity = updates.Severity
			}
			if updates.Condition.Type != "" {
				e.rules[i].Condition = updates.Condition
			}
			if updates.Actions != nil {
				e.rules[i].Actions = updates.Actions
			}
			e.rules[i].UpdatedAt = time.Now()
			return &e.rules[i], nil
		}
	}

	return nil, fmt.Errorf("rule not found: %s", id)
}

// DeleteRule removes the rule with the given ID.
func (e *Engine) DeleteRule(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.rules {
		if e.rules[i].ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", id)
}

// GetRules returns rules matching the optional filter criteria.
func (e *Engine) GetRules(severity string, enabled *bool) []AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []AlertRule
	for _, r := range e.rules {
		if severity != "" && !strings.EqualFold(r.Severity, severity) {
			continue
		}
		if enabled != nil && r.Enabled != *enabled {
			continue
		}
		result = append(result, r)
	}

	return result
}

// GetRule returns the rule with the given ID, or nil if not found.
func (e *Engine) GetRule(id string) *AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for i := range e.rules {
		if e.rules[i].ID == id {
			return &e.rules[i]
		}
	}

	return nil
}

// EvaluateRules evaluates all enabled rules against the provided data and returns triggered alerts.
func (e *Engine) EvaluateRules(data map[string]interface{}) []AlertRecord {
	e.mu.RLock()
	rules := make([]AlertRule, len(e.rules))
	copy(rules, e.rules)
	e.mu.RUnlock()

	var triggered []AlertRecord
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if alert := e.evaluateRule(rule, data); alert != nil {
			record := e.TriggerAlert(*alert)
			triggered = append(triggered, *record)
		}
	}

	return triggered
}

// evaluateRule checks a single rule against the data and returns an alert if the condition is met.
func (e *Engine) evaluateRule(rule AlertRule, data map[string]interface{}) *AlertRecord {
	cond := rule.Condition

	value, ok := data[cond.Field]
	if !ok {
		return nil
	}

	triggered := compareValues(value, cond.Threshold, cond.Operator)
	if !triggered {
		return nil
	}

	return &AlertRecord{
		ID:          e.generateID(),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		Value:       value,
		Threshold:   cond.Threshold,
		Operator:    cond.Operator,
		TriggeredAt: time.Now(),
	}
}

// TriggerAlert records an alert in history and the active alerts map, capping history at 1000.
func (e *Engine) TriggerAlert(alert AlertRecord) *AlertRecord {
	e.mu.Lock()
	defer e.mu.Unlock()

	if alert.ID == "" {
		alert.ID = e.generateID()
	}
	if alert.TriggeredAt.IsZero() {
		alert.TriggeredAt = time.Now()
	}

	e.history = append(e.history, alert)
	if len(e.history) > 1000 {
		e.history = e.history[len(e.history)-1000:]
	}

	e.activeAlerts[alert.ID] = &e.history[len(e.history)-1]
	return e.activeAlerts[alert.ID]
}

// AcknowledgeAlert marks the alert with the given ID as acknowledged.
func (e *Engine) AcknowledgeAlert(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, ok := e.activeAlerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	alert.Acknowledged = true
	now := time.Now()
	alert.AcknowledgedAt = &now

	e.updateHistoryRecord(id, alert)
	return nil
}

// ResolveAlert marks the alert with the given ID as resolved.
func (e *Engine) ResolveAlert(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, ok := e.activeAlerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	alert.Resolved = true
	now := time.Now()
	alert.ResolvedAt = &now

	e.updateHistoryRecord(id, alert)
	delete(e.activeAlerts, id)
	return nil
}

// GetAlertHistory returns alert records matching the given filter.
func (e *Engine) GetAlertHistory(filter AlertFilter) []AlertRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []AlertRecord
	for _, a := range e.history {
		if filter.Severity != "" && !strings.EqualFold(a.Severity, filter.Severity) {
			continue
		}
		if filter.Acknowledged != nil && a.Acknowledged != *filter.Acknowledged {
			continue
		}
		if filter.Resolved != nil && a.Resolved != *filter.Resolved {
			continue
		}
		result = append(result, a)
	}

	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[len(result)-filter.Limit:]
	}

	return result
}

// GetActiveAlerts returns all currently active (unresolved) alerts.
func (e *Engine) GetActiveAlerts() []AlertRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	alerts := make([]AlertRecord, 0, len(e.activeAlerts))
	for _, a := range e.activeAlerts {
		alerts = append(alerts, *a)
	}

	return alerts
}

// getDefaultRules returns the 8 default alert rules.
func (e *Engine) getDefaultRules() []AlertRule {
	now := time.Now()
	return []AlertRule{
		{
			ID:          "pod-crash-looping",
			Name:        "Pod Crash Looping",
			Description: "A pod is crash looping with frequent restarts",
			Enabled:     true,
			Severity:    "critical",
			Condition:   RuleCondition{Type: "pod", Field: "restartCount", Operator: ">", Threshold: 5.0, TimeWindow: "5m"},
			Actions:     []string{"notify"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "high-cpu",
			Name:        "High CPU Usage",
			Description: "Container CPU usage exceeds threshold",
			Enabled:     true,
			Severity:    "warning",
			Condition:   RuleCondition{Type: "pod", Field: "cpuPercent", Operator: ">", Threshold: 80.0, TimeWindow: "10m"},
			Actions:     []string{"notify"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "high-memory",
			Name:        "High Memory Usage",
			Description: "Container memory usage exceeds threshold",
			Enabled:     true,
			Severity:    "warning",
			Condition:   RuleCondition{Type: "pod", Field: "memoryPercent", Operator: ">", Threshold: 85.0, TimeWindow: "10m"},
			Actions:     []string{"notify"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "node-not-ready",
			Name:        "Node Not Ready",
			Description: "A node is in NotReady state",
			Enabled:     true,
			Severity:    "critical",
			Condition:   RuleCondition{Type: "node", Field: "status", Operator: "==", Threshold: "NotReady", TimeWindow: "5m"},
			Actions:     []string{"notify", "page"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "disk-pressure",
			Name:        "Disk Pressure",
			Description: "A node is experiencing disk pressure",
			Enabled:     true,
			Severity:    "critical",
			Condition:   RuleCondition{Type: "node", Field: "diskPressure", Operator: "==", Threshold: true, TimeWindow: "5m"},
			Actions:     []string{"notify"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "pod-pending",
			Name:        "Pod Stuck Pending",
			Description: "A pod has been in Pending state for too long",
			Enabled:     true,
			Severity:    "warning",
			Condition:   RuleCondition{Type: "pod", Field: "pendingSeconds", Operator: ">", Threshold: 300.0, TimeWindow: "10m"},
			Actions:     []string{"notify"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "failed-events",
			Name:        "Failed Events Detected",
			Description: "Warning or Error events detected in the cluster",
			Enabled:     true,
			Severity:    "warning",
			Condition:   RuleCondition{Type: "event", Field: "reason", Operator: "contains", Threshold: "Failed", TimeWindow: "5m"},
			Actions:     []string{"notify"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "image-pull-backoff",
			Name:        "Image Pull Backoff",
			Description: "A pod is stuck in ImagePullBackOff",
			Enabled:     true,
			Severity:    "warning",
			Condition:   RuleCondition{Type: "pod", Field: "status", Operator: "==", Threshold: "ImagePullBackOff", TimeWindow: "5m"},
			Actions:     []string{"notify"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

// generateID produces a unique identifier based on the current Unix nanosecond timestamp.
func (e *Engine) generateID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}

// updateHistoryRecord updates the matching entry in the history slice.
func (e *Engine) updateHistoryRecord(id string, alert *AlertRecord) {
	for i := range e.history {
		if e.history[i].ID == id {
			e.history[i] = *alert
			return
		}
	}
}

// compareValues compares two values using the given operator.
func compareValues(value, threshold interface{}, operator string) bool {
	vf, vok := toFloat64(value)
	tf, tok := toFloat64(threshold)

	switch operator {
	case ">":
		if vok && tok {
			return vf > tf
		}
	case "<":
		if vok && tok {
			return vf < tf
		}
	case ">=":
		if vok && tok {
			return vf >= tf
		}
	case "<=":
		if vok && tok {
			return vf <= tf
		}
	case "==":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", threshold)
	case "!=":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", threshold)
	case "contains":
		return strings.Contains(
			strings.ToLower(fmt.Sprintf("%v", value)),
			strings.ToLower(fmt.Sprintf("%v", threshold)),
		)
	}

	return false
}

// toFloat64 attempts to convert an interface{} to float64 for numeric comparison.
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
