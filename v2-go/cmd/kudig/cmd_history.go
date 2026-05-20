package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kudig/kudig/pkg/history"
	"github.com/kudig/kudig/pkg/types"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage diagnostic history",
	Long: `View and compare diagnostic history entries.

History is stored in ~/.kudig/history/ and includes:
- Timestamp of diagnosis
- Hostname
- Issues found
- Summary statistics`,
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List diagnostic history entries",
	Long:  `List all stored diagnostic history entries, sorted by timestamp (newest first).`,
	RunE:  runHistoryList,
}

var historyDiffCmd = &cobra.Command{
	Use:   "diff <id1> <id2>",
	Short: "Compare two diagnostic history entries",
	Long: `Compare two diagnostic history entries and show the differences.

Arguments:
  id1 - ID of the first history entry
  id2 - ID of the second history entry

Example:
  kudig history diff abc123 def456`,
	Args: cobra.ExactArgs(2),
	RunE: runHistoryDiff,
}

func runHistoryList(_ *cobra.Command, _ []string) error {
	mgr, err := history.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create history manager: %w", err)
	}

	entries, err := mgr.List()
	if err != nil {
		return fmt.Errorf("failed to list history: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No history entries found.")
		fmt.Println("Run a diagnosis first to create history entries.")
		return nil
	}

	fmt.Println("Diagnostic History:")
	fmt.Println("===================")
	fmt.Printf("%-18s %-20s %-15s %-10s %-10s %-10s\n", "ID", "Timestamp", "Hostname", "Critical", "Warning", "Info")
	fmt.Println(strings.Repeat("-", 90))

	for _, entry := range entries {
		shortID := entry.ID
		if len(shortID) > 16 {
			shortID = shortID[:16]
		}
		fmt.Printf("%-18s %-20s %-15s %-10d %-10d %-10d\n",
			shortID,
			entry.Timestamp.Format("2006-01-02 15:04"),
			truncate(entry.Hostname, 15),
			entry.Summary.Critical,
			entry.Summary.Warning,
			entry.Summary.Info,
		)
	}

	fmt.Printf("\nTotal entries: %d\n", len(entries))
fmt.Println("\\nUse 'kudig history diff <id1> <id2>' to compare entries.")
	return nil
}

func runHistoryDiff(_ *cobra.Command, args []string) error {
	id1, id2 := args[0], args[1]

	mgr, err := history.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create history manager: %w", err)
	}

	diff, err := mgr.Diff(id1, id2)
	if err != nil {
		return fmt.Errorf("failed to diff history entries: %w", err)
	}

	fmt.Println("History Comparison:")
	fmt.Println("===================")
	fmt.Printf("Entry 1: %s (%s on %s)\n", diff.Entry1.ID[:16], diff.Entry1.Mode, diff.Entry1.Hostname)
	fmt.Printf("         %s\n", diff.Entry1.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Entry 2: %s (%s on %s)\n", diff.Entry2.ID[:16], diff.Entry2.Mode, diff.Entry2.Hostname)
	fmt.Printf("         %s\n", diff.Entry2.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	fmt.Println("Summary Changes:")
	fmt.Printf("  Critical: %d -> %d (%+d)\n", diff.Entry1.Summary.Critical, diff.Entry2.Summary.Critical, diff.Entry2.Summary.Critical-diff.Entry1.Summary.Critical)
	fmt.Printf("  Warning:  %d -> %d (%+d)\n", diff.Entry1.Summary.Warning, diff.Entry2.Summary.Warning, diff.Entry2.Summary.Warning-diff.Entry1.Summary.Warning)
	fmt.Printf("  Info:     %d -> %d (%+d)\n", diff.Entry1.Summary.Info, diff.Entry2.Summary.Info, diff.Entry2.Summary.Info-diff.Entry1.Summary.Info)
	fmt.Println()

	if len(diff.AddedIssues) > 0 {
		fmt.Printf("Added Issues (%d):\n", len(diff.AddedIssues))
		fmt.Println(strings.Repeat("-", 40))
		for _, issue := range diff.AddedIssues {
			fmt.Printf("  [%s] %s\n", severityString(issue.Severity), issue.CNName)
			fmt.Printf("      %s\n", issue.Details)
			fmt.Println()
		}
	}

	if len(diff.RemovedIssues) > 0 {
		fmt.Printf("Resolved Issues (%d):\n", len(diff.RemovedIssues))
		fmt.Println(strings.Repeat("-", 40))
		for _, issue := range diff.RemovedIssues {
			fmt.Printf("  [%s] %s\n", severityString(issue.Severity), issue.CNName)
			fmt.Printf("      %s\n", issue.Details)
			fmt.Println()
		}
	}

	if len(diff.AddedIssues) == 0 && len(diff.RemovedIssues) == 0 {
		fmt.Println("No changes detected between the two entries.")
	}
	return nil
}

func severityString(s types.Severity) string {
	switch s {
	case types.SeverityCritical:
		return "CRITICAL"
	case types.SeverityWarning:
		return "WARNING"
	case types.SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}
