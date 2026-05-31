// Package loganalysis provides Kubernetes pod log analysis, including
// error/warning classification, stack trace extraction, security event
// detection, performance metric parsing, database query detection, and
// API call extraction.
package loganalysis

import "time"

// LogAnalysis holds the results of analyzing a block of log text.
type LogAnalysis struct {
	TotalLines         int              `json:"totalLines"`
	ErrorCount         int              `json:"errorCount"`
	WarningCount       int              `json:"warningCount"`
	InfoCount          int              `json:"infoCount"`
	DebugCount         int              `json:"debugCount"`
	Errors             []LogEntry       `json:"errors,omitempty"`
	Warnings           []LogEntry       `json:"warnings,omitempty"`
	StackTraces        []string         `json:"stackTraces,omitempty"`
	PerformanceMetrics PerformanceData  `json:"performanceMetrics"`
	SecurityEvents     []SecurityEvent  `json:"securityEvents,omitempty"`
	DatabaseQueries    []DatabaseQuery  `json:"databaseQueries,omitempty"`
	APICalls           []APICall        `json:"apiCalls,omitempty"`
	TimeDistribution   map[string]int   `json:"timeDistribution"`
	Recommendations    []Recommendation `json:"recommendations,omitempty"`
	LogLevels          map[string]int   `json:"logLevels"`
	PatternStats       map[string]int   `json:"patternStats"`
}

// LogEntry represents a single classified log line.
type LogEntry struct {
	Line      int       `json:"line"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
}

// PerformanceData holds performance-related metrics extracted from logs.
type PerformanceData struct {
	SlowRequests      []SlowRequest     `json:"slowRequests,omitempty"`
	ResponseTimeStats ResponseTimeStats `json:"responseTimeStats"`
}

// SlowRequest represents a request that exceeded a latency threshold.
type SlowRequest struct {
	URL          string `json:"url"`
	ResponseTime string `json:"responseTime"`
}

// ResponseTimeStats aggregates response-time measurements.
type ResponseTimeStats struct {
	MinMs float64 `json:"minMs"`
	MaxMs float64 `json:"maxMs"`
	AvgMs float64 `json:"avgMs"`
	Count int     `json:"count"`
}

// SecurityEvent represents a security-relevant log entry.
type SecurityEvent struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// DatabaseQuery captures SQL-like statements detected in logs.
type DatabaseQuery struct {
	Type  string `json:"type"`
	Query string `json:"query"`
}

// APICall captures HTTP API calls detected in logs.
type APICall struct {
	Method     string `json:"method"`
	URL        string `json:"url"`
	StatusCode int    `json:"statusCode"`
}

// Recommendation is an actionable suggestion generated from analysis results.
type Recommendation struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}
