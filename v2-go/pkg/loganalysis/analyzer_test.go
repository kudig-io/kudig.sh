package loganalysis

import (
	"strings"
	"testing"
)

func TestAnalyzer_AllLevels(t *testing.T) {
	logs := `2024-01-15T10:00:00Z INFO  application started
2024-01-15T10:01:00Z DEBUG processing request id=123
2024-01-15T10:02:00Z TRACE entering function
2024-01-15T10:03:00Z WARN  deprecated API usage
2024-01-15T10:04:00Z ERROR connection refused to db
2024-01-15T10:05:00Z FATAL out of memory
`
	a := NewAnalyzer()
	result := a.AnalyzeLogs(logs)

	if result.TotalLines != 7 {
		t.Errorf("expected 7 total lines, got %d", result.TotalLines)
	}
	if result.InfoCount != 1 {
		t.Errorf("expected 1 info, got %d", result.InfoCount)
	}
	if result.DebugCount != 1 {
		t.Errorf("expected 1 debug, got %d", result.DebugCount)
	}
	if result.ErrorCount != 2 { // error + fatal
		t.Errorf("expected 2 errors, got %d", result.ErrorCount)
	}
	if result.WarningCount != 1 {
		t.Errorf("expected 1 warning, got %d", result.WarningCount)
	}
	if result.LogLevels["fatal"] != 1 {
		t.Errorf("expected 1 fatal level, got %d", result.LogLevels["fatal"])
	}
	if result.LogLevels["trace"] != 1 {
		t.Errorf("expected 1 trace level, got %d", result.LogLevels["trace"])
	}
}

func TestAnalyzer_SecurityEvents(t *testing.T) {
	logs := `2024-01-15T10:00:00Z WARN  unauthorized access attempt from 10.0.0.1
2024-01-15T10:01:00Z ERROR permission denied for user admin
2024-01-15T10:02:00Z INFO  normal operation
2024-01-15T10:03:00Z ERROR SQL injection attempt detected
`
	a := NewAnalyzer()
	result := a.AnalyzeLogs(logs)

	if len(result.SecurityEvents) != 3 {
		t.Fatalf("expected 3 security events, got %d", len(result.SecurityEvents))
	}

	expected := []struct {
		typ      string
		severity string
	}{
		{"Unauthorized Access", "high"},
		{"Permission Denied", "medium"},
		{"SQL Injection", "high"},
	}
	for i, exp := range expected {
		if result.SecurityEvents[i].Type != exp.typ {
			t.Errorf("event %d: expected type %q, got %q", i, exp.typ, result.SecurityEvents[i].Type)
		}
		if result.SecurityEvents[i].Severity != exp.severity {
			t.Errorf("event %d: expected severity %q, got %q", i, exp.severity, result.SecurityEvents[i].Severity)
		}
	}
}

func TestAnalyzer_DatabaseQueries(t *testing.T) {
	logs := `2024-01-15T10:00:00Z INFO  Executing query: SELECT * FROM users WHERE id = 1
2024-01-15T10:01:00Z INFO  UPDATE users SET last_login = NOW() WHERE id = 1
2024-01-15T10:02:00Z INFO  DELETE FROM sessions WHERE expired = true
2024-01-15T10:03:00Z INFO  normal log line
`
	a := NewAnalyzer()
	result := a.AnalyzeLogs(logs)

	if len(result.DatabaseQueries) != 3 {
		t.Fatalf("expected 3 database queries, got %d", len(result.DatabaseQueries))
	}

	expectedTypes := []string{"SELECT", "UPDATE", "DELETE"}
	for i, exp := range expectedTypes {
		if result.DatabaseQueries[i].Type != exp {
			t.Errorf("query %d: expected type %q, got %q", i, exp, result.DatabaseQueries[i].Type)
		}
	}
}

func TestAnalyzer_APICalls(t *testing.T) {
	logs := `2024-01-15T10:00:00Z INFO  POST https://api.example.com/v1/users 201
2024-01-15T10:01:00Z INFO  GET https://api.example.com/v1/users/123 200
2024-01-15T10:02:00Z ERROR DELETE https://api.example.com/v1/users/456 404
2024-01-15T10:03:00Z INFO  normal line
`
	a := NewAnalyzer()
	result := a.AnalyzeLogs(logs)

	if len(result.APICalls) != 3 {
		t.Fatalf("expected 3 API calls, got %d", len(result.APICalls))
	}

	expected := []struct {
		method string
		status int
	}{
		{"POST", 201},
		{"GET", 200},
		{"DELETE", 404},
	}
	for i, exp := range expected {
		if result.APICalls[i].Method != exp.method {
			t.Errorf("call %d: expected method %q, got %q", i, exp.method, result.APICalls[i].Method)
		}
		if result.APICalls[i].StatusCode != exp.status {
			t.Errorf("call %d: expected status %d, got %d", i, exp.status, result.APICalls[i].StatusCode)
		}
	}
}

func TestAnalyzer_PerformanceMetrics(t *testing.T) {
	logs := `2024-01-15T10:00:00Z INFO  request duration: 150ms
2024-01-15T10:01:00Z INFO  request took 2.5s
2024-01-15T10:02:00Z INFO  elapsed time: 800ms
2024-01-15T10:03:00Z INFO  request duration: 3s
2024-01-15T10:04:00Z INFO  normal line
`
	a := NewAnalyzer()
	result := a.AnalyzeLogs(logs)

	// Slow requests: >1000ms => 2.5s and 3s => 2 slow requests.
	if len(result.PerformanceMetrics.SlowRequests) != 2 {
		t.Errorf("expected 2 slow requests, got %d", len(result.PerformanceMetrics.SlowRequests))
	}

	stats := result.PerformanceMetrics.ResponseTimeStats
	if stats.Count != 4 {
		t.Errorf("expected 4 duration measurements, got %d", stats.Count)
	}
	if stats.MinMs != 150 {
		t.Errorf("expected min 150ms, got %f", stats.MinMs)
	}
	if stats.MaxMs != 3000 {
		t.Errorf("expected max 3000ms, got %f", stats.MaxMs)
	}
	if stats.AvgMs != 1612.5 {
		t.Errorf("expected avg 1612.5ms, got %f", stats.AvgMs)
	}
}

func TestAnalyzer_TimeDistribution(t *testing.T) {
	logs := `2024-01-15T08:30:00Z INFO  morning event
2024-01-15T08:45:00Z INFO  morning event 2
2024-01-15T14:20:00Z INFO  afternoon event
2024-01-15T22:10:00Z ERROR night event
`
	a := NewAnalyzer()
	result := a.AnalyzeLogs(logs)

	if result.TimeDistribution["08:00-08:59"] != 2 {
		t.Errorf("expected 2 entries in 08:00-08:59, got %d", result.TimeDistribution["08:00-08:59"])
	}
	if result.TimeDistribution["14:00-14:59"] != 1 {
		t.Errorf("expected 1 entry in 14:00-14:59, got %d", result.TimeDistribution["14:00-14:59"])
	}
	if result.TimeDistribution["22:00-22:59"] != 1 {
		t.Errorf("expected 1 entry in 22:00-22:59, got %d", result.TimeDistribution["22:00-22:59"])
	}
}

func TestAnalyzer_StackTraces(t *testing.T) {
	logs := `2024-01-15T10:00:00Z ERROR Exception in thread "main" java.lang.NullPointerException
2024-01-15T10:01:00Z ERROR 	at com.example.Main.main(Main.java:15)
2024-01-15T10:02:00Z panic: runtime error: index out of range
2024-01-15T10:03:00Z goroutine 1 [running]:
2024-01-15T10:04:00Z INFO  normal line
`
	a := NewAnalyzer()
	result := a.AnalyzeLogs(logs)

	if len(result.StackTraces) != 4 {
		t.Errorf("expected 4 stack-trace lines, got %d", len(result.StackTraces))
	}
}

func TestAnalyzer_Recommendations(t *testing.T) {
	// Build a log with many errors, warnings, slow requests, security events, timeouts.
	var sb strings.Builder
	for i := 0; i < 15; i++ {
		sb.WriteString("2024-01-15T10:00:00Z ERROR connection timeout\n")
	}
	for i := 0; i < 25; i++ {
		sb.WriteString("2024-01-15T10:00:00Z WARN  deprecated call\n")
	}
	for i := 0; i < 8; i++ {
		sb.WriteString("2024-01-15T10:00:00Z INFO  request took 2.0s\n")
	}
	sb.WriteString("2024-01-15T10:00:00Z ERROR unauthorized access\n")
	sb.WriteString("2024-01-15T10:00:00Z ERROR stack trace: at com.example.Foo.bar(Foo.java:10)\n")
	for i := 0; i < 8; i++ {
		sb.WriteString("2024-01-15T10:00:00Z ERROR retry attempt failed\n")
	}

	a := NewAnalyzer()
	result := a.AnalyzeLogs(sb.String())

	if len(result.Recommendations) == 0 {
		t.Fatal("expected recommendations, got none")
	}

	// Spot-check a few recommendation types.
	var hasErrorRec, hasSecurityRec, hasStackRec, hasSlowRec bool
	for _, r := range result.Recommendations {
		if strings.Contains(r.Message, "errors") {
			hasErrorRec = true
		}
		if strings.Contains(r.Message, "security") {
			hasSecurityRec = true
		}
		if strings.Contains(r.Message, "stack-trace") {
			hasStackRec = true
		}
		if strings.Contains(r.Message, "slow requests") {
			hasSlowRec = true
		}
	}

	if !hasErrorRec {
		t.Error("missing error-count recommendation")
	}
	if !hasSecurityRec {
		t.Error("missing security-event recommendation")
	}
	if !hasStackRec {
		t.Error("missing stack-trace recommendation")
	}
	if !hasSlowRec {
		t.Error("missing slow-request recommendation")
	}
}

func TestAnalyzer_HTTPStatusPatternStats(t *testing.T) {
	logs := `2024-01-15T10:00:00Z INFO  HTTP/1.1 200 OK
2024-01-15T10:01:00Z INFO  HTTP/1.1 404 Not Found
2024-01-15T10:02:00Z INFO  HTTP/1.1 500 Internal Server Error
2024-01-15T10:03:00Z INFO  HTTP/1.1 200 OK
`
	a := NewAnalyzer()
	result := a.AnalyzeLogs(logs)

	if result.PatternStats["http_200"] != 2 {
		t.Errorf("expected http_200=2, got %d", result.PatternStats["http_200"])
	}
	if result.PatternStats["http_404"] != 1 {
		t.Errorf("expected http_404=1, got %d", result.PatternStats["http_404"])
	}
	if result.PatternStats["http_500"] != 1 {
		t.Errorf("expected http_500=1, got %d", result.PatternStats["http_500"])
	}
}

func TestAnalyzer_IPAndURLStats(t *testing.T) {
	logs := `2024-01-15T10:00:00Z INFO  client connected from 192.168.1.1
2024-01-15T10:01:00Z INFO  forwarding to https://backend.internal/api
2024-01-15T10:02:00Z INFO  client connected from 10.0.0.5
`
	a := NewAnalyzer()
	result := a.AnalyzeLogs(logs)

	if result.PatternStats["ip_addresses"] != 2 {
		t.Errorf("expected ip_addresses=2, got %d", result.PatternStats["ip_addresses"])
	}
	if result.PatternStats["urls"] != 1 {
		t.Errorf("expected urls=1, got %d", result.PatternStats["urls"])
	}
}
