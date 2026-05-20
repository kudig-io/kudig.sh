package alerting

import "time"

// AlertRule defines a monitoring rule that triggers alerts when conditions are met.
type AlertRule struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Enabled     bool            `json:"enabled"`
	Severity    string          `json:"severity"`
	Condition   RuleCondition   `json:"condition"`
	Actions     []string        `json:"actions"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// RuleCondition specifies the criteria for evaluating a rule against cluster data.
type RuleCondition struct {
	Type       string      `json:"type"`       // pod, node, event
	Field      string      `json:"field"`      // field path to evaluate
	Operator   string      `json:"operator"`   // >, <, >=, <=, ==, !=, contains
	Threshold  interface{} `json:"threshold"`  // comparison value
	TimeWindow string      `json:"timeWindow"` // evaluation time window (e.g. "5m")
}

// AlertRecord represents a triggered alert instance.
type AlertRecord struct {
	ID             string     `json:"id"`
	RuleID         string     `json:"ruleId"`
	RuleName       string     `json:"ruleName"`
	Severity       string     `json:"severity"`
	Value          interface{} `json:"value"`
	Threshold      interface{} `json:"threshold"`
	Operator       string     `json:"operator"`
	TriggeredAt    time.Time  `json:"triggeredAt"`
	Acknowledged   bool       `json:"acknowledged"`
	Resolved       bool       `json:"resolved"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
}

// AlertFilter provides criteria for filtering alert queries.
type AlertFilter struct {
	Severity      string `json:"severity,omitempty"`
	Acknowledged  *bool  `json:"acknowledged,omitempty"`
	Resolved      *bool  `json:"resolved,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}
