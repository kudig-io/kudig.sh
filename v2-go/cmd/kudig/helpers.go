package main

import (
	"fmt"
	"os"

	"github.com/kudig/kudig/pkg/notifier"
	"github.com/kudig/kudig/pkg/types"
)

// writeOutput writes output to file or stdout based on global flags.
func writeOutput(output []byte) error {
	if outputFile != "" {
		if err := os.WriteFile(outputFile, output, 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "报告已保存到: %s\n", outputFile)
		}
	} else {
		fmt.Println(string(output))
	}
	return nil
}

// severityExitCode returns appropriate exit code based on issue severity.
func severityExitCode(issues []types.Issue, status *string) error {
	maxSev := types.MaxSeverity(issues)
	if maxSev == types.SeverityCritical {
		*status = "critical_issues"
		return &exitError{code: 2}
	} else if len(issues) > 0 {
		*status = "issues_found"
		return &exitError{code: 1}
	}
	return nil
}

// sendNotification sends webhook notifications if configured and issues meet severity threshold.
func sendNotification(hostname, mode string, issues []types.Issue) {
	notifyConfig := notifier.NewConfigFromEnv()
	if !notifyConfig.ShouldNotify(issues) {
		return
	}

	multiNotifier := notifier.NewMultiNotifier(notifyConfig)
	if multiNotifier == nil || len(multiNotifier.Notifiers) == 0 {
		return
	}

	title := fmt.Sprintf("Kudig Alert: Issues detected on %s", hostname)
	message := fmt.Sprintf("Diagnostic mode: %s\nFound %d issues (%d critical, %d warning, %d info)",
		mode,
		len(issues),
		countBySeverity(issues, types.SeverityCritical),
		countBySeverity(issues, types.SeverityWarning),
		countBySeverity(issues, types.SeverityInfo),
	)

	errors := multiNotifier.Send(title, message, issues)
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "Notification error: %v\n", err)
		}
	}
}

// countBySeverity counts issues matching the given severity.
func countBySeverity(issues []types.Issue, sev types.Severity) int {
	count := 0
	for _, issue := range issues {
		if issue.Severity == sev {
			count++
		}
	}
	return count
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
