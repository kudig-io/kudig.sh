package loganalysis

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Pre-compiled regular expressions for performance.
var (
	// durationRe matches duration patterns like "duration: 123ms", "took 1.5s"
	durationRe = regexp.MustCompile(`(?i)(?:duration|elapsed|time|took|spent)[\s:]+(\d+(?:\.\d+)?)\s*(ms|milliseconds|s|seconds|sec)`)

	// dbQueryRe matches SQL-like statements (including UPDATE <table> SET).
	dbQueryRe = regexp.MustCompile(`(?i)(select|insert|update|delete|create|drop|alter)(?:\s+(?:\*?\s*from|into|table|index|database)|\s+\w+\s+set)`)

	// apiCallRe matches HTTP method + URL patterns.
	apiCallRe = regexp.MustCompile(`(?i)(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s+(https?://\S+)`)

	// httpStatusRe matches HTTP status codes (200, 404, 500, etc.).
	httpStatusRe = regexp.MustCompile(`\b(\d{3})\b`)

	// httpResponseLineRe matches HTTP/1.1 200 style status lines.
	httpResponseLineRe = regexp.MustCompile(`HTTP/\d+\.\d+\s+(\d{3})`)

	// ipRe matches IPv4 addresses.
	ipRe = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)

	// urlRe matches URLs.
	urlRe = regexp.MustCompile(`https?://\S+`)

	// timestampRe matches ISO-8601-like timestamps.
	timestampRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?`)

	// hourExtractRe extracts the hour from a timestamp.
	hourExtractRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ](\d{2}):\d{2}:\d{2}`)
)

// Analyzer performs static analysis on raw log text.
type Analyzer struct{}

// NewAnalyzer returns a new Analyzer instance.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// AnalyzeLogs parses the supplied log text and returns a structured analysis.
func (a *Analyzer) AnalyzeLogs(logs string) *LogAnalysis {
	analysis := &LogAnalysis{
		LogLevels:        make(map[string]int),
		PatternStats:     make(map[string]int),
		TimeDistribution: make(map[string]int),
	}

	lines := strings.Split(logs, "\n")
	analysis.TotalLines = len(lines)

	var slowRequestDurations []float64 // durations in milliseconds

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		lineNum := i + 1
		lower := strings.ToLower(line)

		// Classify log level and count.
		level := classifyLine(lower)
		analysis.LogLevels[level]++

		entry := LogEntry{
			Line:    lineNum,
			Content: line,
			Level:   level,
		}

		// Try to parse an ISO-8601 timestamp prefix.
		if ts, ok := extractTimestamp(line); ok {
			entry.Timestamp = ts
		}

		// Categorize by level.
		switch level {
		case "fatal":
			// Fatal is treated as a critical error.
			analysis.ErrorCount++
			analysis.Errors = append(analysis.Errors, entry)
		case "error":
			analysis.ErrorCount++
			analysis.Errors = append(analysis.Errors, entry)
		case "warning":
			analysis.WarningCount++
			analysis.Warnings = append(analysis.Warnings, entry)
		case "info":
			analysis.InfoCount++
		case "debug":
			analysis.DebugCount++
		}

		// Stack trace detection (Go, Java, Python, JavaScript).
		if isStackTrace(line) {
			analysis.StackTraces = append(analysis.StackTraces, line)
		}

		// Security event detection.
		if evt, ok := detectSecurityEvent(lower, line); ok {
			analysis.SecurityEvents = append(analysis.SecurityEvents, evt)
		}

		// Performance / slow-request detection.
		if durationMs := extractDurationMs(line); durationMs > 0 {
			if durationMs > 1000 {
				analysis.PerformanceMetrics.SlowRequests = append(
					analysis.PerformanceMetrics.SlowRequests,
					SlowRequest{
						URL:          line,
						ResponseTime: formatDuration(durationMs),
					},
				)
			}
			slowRequestDurations = append(slowRequestDurations, durationMs)
		}

		// Database query detection.
		if dbq, ok := detectDatabaseQuery(line); ok {
			analysis.DatabaseQueries = append(analysis.DatabaseQueries, dbq)
		}

		// API call detection.
		if api, ok := detectAPICall(line); ok {
			analysis.APICalls = append(analysis.APICalls, api)
		}

		// Pattern statistics.
		countPatterns(lower, analysis.PatternStats)

		// HTTP status code analysis.
		countHTTPStatus(line, analysis.PatternStats)

		// IP address analysis.
		if ipRe.MatchString(line) {
			analysis.PatternStats["ip_addresses"]++
		}

		// URL analysis.
		if urlRe.MatchString(line) {
			analysis.PatternStats["urls"]++
		}

		// Time distribution by hour.
		countTimeDistribution(line, analysis.TimeDistribution)
	}

	// Compute response-time statistics.
	analysis.PerformanceMetrics.ResponseTimeStats = computeResponseTimeStats(slowRequestDurations)

	// Generate actionable recommendations.
	analysis.Recommendations = generateRecommendations(analysis)

	return analysis
}

// classifyLine returns a log level string based on keyword presence.
func classifyLine(lower string) string {
	switch {
	case strings.Contains(lower, "fatal"):
		return "fatal"
	case strings.Contains(lower, "error") || strings.Contains(lower, "panic"):
		return "error"
	case strings.Contains(lower, "warn"):
		return "warning"
	case strings.Contains(lower, "debug"):
		return "debug"
	case strings.Contains(lower, "trace"):
		return "trace"
	default:
		return "info"
	}
}

// isStackTrace checks whether the line looks like part of a stack trace.
func isStackTrace(line string) bool {
	patterns := []string{
		"goroutine",
		"panic:",
		"Traceback (most recent call last)",
		"\tat ",
		"stack trace",
		" Caused by: ",
		".java:",
		".go:",
		"Exception in thread",
		"java.lang.",
	}
	for _, p := range patterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	// JavaScript / Java style: "at " + "("
	if strings.Contains(line, "at ") && strings.Contains(line, "(") {
		return true
	}
	return false
}

// detectSecurityEvent checks for security-relevant keywords.
func detectSecurityEvent(lower, original string) (SecurityEvent, bool) {
	keywords := []struct {
		keyword  string
		evtType  string
		severity string
	}{
		{"unauthorized", "Unauthorized Access", "high"},
		{"forbidden", "Forbidden Access", "high"},
		{"auth failed", "Authentication Failure", "medium"},
		{"authentication failure", "Authentication Failure", "medium"},
		{"access denied", "Access Denied", "high"},
		{"invalid token", "Invalid Token", "medium"},
		{"permission denied", "Permission Denied", "medium"},
		{"security", "Security Event", "medium"},
		{"xss", "XSS Attempt", "high"},
		{"sql injection", "SQL Injection", "high"},
		{"csrf", "CSRF Attempt", "high"},
		{"brute force", "Brute Force", "high"},
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw.keyword) {
			return SecurityEvent{
				Type:     kw.evtType,
				Message:  original,
				Severity: kw.severity,
			}, true
		}
	}
	return SecurityEvent{}, false
}

// extractDurationMs scans a line for duration indicators and returns the value in milliseconds.
func extractDurationMs(line string) float64 {
	matches := durationRe.FindStringSubmatch(line)
	if len(matches) < 3 {
		return 0
	}
	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}
	unit := strings.ToLower(matches[2])
	if strings.HasPrefix(unit, "s") {
		return val * 1000
	}
	return val
}

// formatDuration converts milliseconds to a human-readable string.
func formatDuration(ms float64) string {
	if ms >= 1000 {
		return strconv.FormatFloat(ms/1000, 'f', 2, 64) + "s"
	}
	return strconv.FormatFloat(ms, 'f', 1, 64) + "ms"
}

// detectDatabaseQuery checks for SQL-like statements.
func detectDatabaseQuery(line string) (DatabaseQuery, bool) {
	matches := dbQueryRe.FindStringSubmatch(line)
	if len(matches) < 2 {
		return DatabaseQuery{}, false
	}
	return DatabaseQuery{
		Type:  strings.ToUpper(matches[1]),
		Query: strings.TrimSpace(line),
	}, true
}

// detectAPICall checks for HTTP method + URL patterns.
func detectAPICall(line string) (APICall, bool) {
	matches := apiCallRe.FindStringSubmatch(line)
	if len(matches) < 3 {
		return APICall{}, false
	}
	status := 0
	// Extract status code from the portion of the line after the URL
	// to avoid matching numeric IDs inside the URL path.
	idx := strings.Index(line, matches[2])
	if idx >= 0 {
		afterURL := line[idx+len(matches[2]):]
		if s := httpStatusRe.FindString(afterURL); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n >= 100 && n < 600 {
				status = n
			}
		}
	}
	return APICall{
		Method:     strings.ToUpper(matches[1]),
		URL:        matches[2],
		StatusCode: status,
	}, true
}

// countPatterns tallies occurrences of common operational keywords.
func countPatterns(lower string, stats map[string]int) {
	patterns := []string{
		"timeout", "retry", "connection refused", "rate limit",
		"oom", "database", "cache", "api", "memory", "cpu",
		"disk", "network", "ssl", "tls",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			stats[p]++
		}
	}
}

// countHTTPStatus detects HTTP response status codes and tallies them.
func countHTTPStatus(line string, stats map[string]int) {
	if matches := httpResponseLineRe.FindStringSubmatch(line); len(matches) >= 2 {
		stats["http_"+matches[1]]++
		return
	}
}

// countTimeDistribution extracts the hour from timestamps and builds a distribution map.
func countTimeDistribution(line string, dist map[string]int) {
	if !timestampRe.MatchString(line) {
		return
	}
	matches := hourExtractRe.FindStringSubmatch(line)
	if len(matches) < 2 {
		return
	}
	hour := matches[1]
	slot := hour + ":00-" + hour + ":59"
	dist[slot]++
}

// computeResponseTimeStats calculates min/max/avg from a slice of millisecond values.
func computeResponseTimeStats(values []float64) ResponseTimeStats {
	if len(values) == 0 {
		return ResponseTimeStats{}
	}
	min := math.MaxFloat64
	max := 0.0
	sum := 0.0
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}
	return ResponseTimeStats{
		MinMs: math.Round(min*100) / 100,
		MaxMs: math.Round(max*100) / 100,
		AvgMs: math.Round(sum/float64(len(values))*100) / 100,
		Count: len(values),
	}
}

// generateRecommendations creates actionable suggestions from analysis results.
func generateRecommendations(a *LogAnalysis) []Recommendation {
	var recs []Recommendation

	if a.ErrorCount > 10 {
		recs = append(recs, Recommendation{
			Severity: "high",
			Message:  "Detected " + strconv.Itoa(a.ErrorCount) + " errors; review application logs and error handling immediately.",
		})
	}

	if a.WarningCount > 20 {
		recs = append(recs, Recommendation{
			Severity: "medium",
			Message:  "Detected " + strconv.Itoa(a.WarningCount) + " warnings; consider optimizing to reduce noise.",
		})
	}

	if len(a.StackTraces) > 0 {
		recs = append(recs, Recommendation{
			Severity: "high",
			Message:  "Detected " + strconv.Itoa(len(a.StackTraces)) + " stack-trace lines; inspect exception handling.",
		})
	}

	if len(a.PerformanceMetrics.SlowRequests) > 5 {
		recs = append(recs, Recommendation{
			Severity: "medium",
			Message:  "Detected " + strconv.Itoa(len(a.PerformanceMetrics.SlowRequests)) + " slow requests (>1s); investigate performance bottlenecks.",
		})
	}

	if len(a.SecurityEvents) > 0 {
		recs = append(recs, Recommendation{
			Severity: "high",
			Message:  "Detected " + strconv.Itoa(len(a.SecurityEvents)) + " security-related events; review security configuration immediately.",
		})
	}

	if a.PatternStats["timeout"] > 5 {
		recs = append(recs, Recommendation{
			Severity: "medium",
			Message:  "Multiple timeout occurrences detected; check network connectivity and timeout settings.",
		})
	}

	if a.PatternStats["retry"] > 5 {
		recs = append(recs, Recommendation{
			Severity: "medium",
			Message:  "Multiple retry occurrences detected; verify dependent service availability.",
		})
	}

	if a.PatternStats["memory"] > 10 {
		recs = append(recs, Recommendation{
			Severity: "medium",
			Message:  "Frequent memory-related log entries; monitor memory usage and check for leaks.",
		})
	}

	return recs
}

// extractTimestamp tries to parse a leading ISO-8601 / common timestamp from a log line.
func extractTimestamp(line string) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"Jan 02 15:04:05",
	}
	// Try to find timestamp prefix up to a reasonable length.
	maxLen := len(line)
	if maxLen > 35 {
		maxLen = 35
	}
	for _, layout := range formats {
		if maxLen < len(layout) {
			continue
		}
		if ts, err := time.Parse(layout, line[:len(layout)]); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}
