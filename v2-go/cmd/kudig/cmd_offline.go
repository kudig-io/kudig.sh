package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kudig/kudig/pkg/analyzer"
	"github.com/kudig/kudig/pkg/collector"
	"github.com/kudig/kudig/pkg/history"
	"github.com/kudig/kudig/pkg/legacy"
	"github.com/kudig/kudig/pkg/metrics"
	"github.com/kudig/kudig/pkg/reporter"
	"github.com/kudig/kudig/pkg/types"
)

var offlineCmd = &cobra.Command{
	Use:   "offline <diagnose_path>",
	Short: "Analyze diagnostic data from a directory",
	Long: `Analyze diagnostic data collected by diagnose_k8s.sh.

Example:
  kudig offline /tmp/diagnose_1702468800
  kudig offline --format json /tmp/diagnose_1702468800`,
	Args:  cobra.ExactArgs(1),
	RunE:  runOffline,
}

var legacyCmd = &cobra.Command{
	Use:   "legacy <diagnose_path>",
	Short: "Run analysis using legacy kudig.sh script",
	Long: `Run the original kudig.sh script for backward compatibility.

Example:
  kudig legacy /tmp/diagnose_1702468800`,
	Args:  cobra.ExactArgs(1),
	RunE:  runLegacy,
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze <diagnose_path>",
	Short: "Analyze diagnostic data (alias for offline)",
	Long:  `Alias for 'kudig offline'. Analyze diagnostic data from a directory.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runOffline,
}

var listAnalyzersCmd = &cobra.Command{
	Use:   "list-analyzers",
	Short: "List all available analyzers",
	RunE:  runListAnalyzers,
}

func runOffline(cmd *cobra.Command, args []string) error {
	diagnosePath := args[0]

	startTime := time.Now()
	status := "success"
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordDiagnosis(types.ModeOffline, status, duration)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if verbose {
		fmt.Fprintf(os.Stderr, "================================================================\n")
		fmt.Fprintf(os.Stderr, "  kudig v%s - Kubernetes Diagnostic Toolkit\n", version)
		fmt.Fprintf(os.Stderr, "================================================================\n\n")
		fmt.Fprintf(os.Stderr, "诊断目录: %s\n", diagnosePath)
		fmt.Fprintf(os.Stderr, "分析时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	}

	col, ok := collector.GetCollector(types.ModeOffline)
	if !ok {
		status = "collector_error"
		return fmt.Errorf("offline collector not available")
	}

	config := collector.NewOfflineConfig(diagnosePath)
	data, err := col.Collect(ctx, config)
	if err != nil {
		status = "collect_error"
		return fmt.Errorf("failed to collect data: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "节点信息: %s\n", data.NodeInfo.Hostname)
		fmt.Fprintf(os.Stderr, "\n开始诊断检查...\n\n")
	}

	results, err := analyzer.DefaultRegistry.ExecuteAll(ctx, data)
	if err != nil {
		status = "analyze_error"
		return fmt.Errorf("failed to run analyzers: %w", err)
	}

	metrics.RecordAnalyzers(types.ModeOffline, len(results))
	issues := analyzer.CollectIssues(results)
	metrics.RecordIssues(issues)

	issues = reporter.DeduplicateIssues(issues)
	issues = reporter.SortIssuesBySeverity(issues)

	metadata := reporter.NewReportMetadata()
	metadata.Hostname = data.NodeInfo.Hostname
	metadata.DiagnosePath = diagnosePath
	metadata.Mode = "offline"

	outputFormat := format
	if jsonFlag, err := cmd.Flags().GetBool("json"); err == nil && jsonFlag {
		outputFormat = "json"
	}

	rep, ok := reporter.GetReporter(outputFormat)
	if !ok {
		return fmt.Errorf("unknown format: %s", outputFormat)
	}

	output, err := rep.Generate(issues, metadata)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	if err := writeOutput(output); err != nil {
		return err
	}

	if histMgr, err := history.NewManager(); err == nil {
		if _, err := histMgr.Save(data.NodeInfo.Hostname, "offline", issues); err == nil && verbose {
			fmt.Fprintf(os.Stderr, "已保存到历史记录\n")
		}
	}

	sendNotification(data.NodeInfo.Hostname, "offline", issues)
	return severityExitCode(issues, &status)
}

func runLegacy(_ *cobra.Command, args []string) error {
	diagnosePath := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	legacyCol, err := legacy.NewLegacyCollector("")
	if err != nil {
		return fmt.Errorf("failed to initialize legacy mode: %w", err)
	}

	report, err := legacyCol.GetReport(ctx, diagnosePath, verbose)
	if err != nil {
		return fmt.Errorf("legacy analysis failed: %w", err)
	}

	issues := legacy.ConvertBashReportToIssues(report)
	metadata := reporter.NewReportMetadata()
	metadata.Hostname = report.Hostname
	metadata.DiagnosePath = report.DiagnoseDir
	metadata.Engine = "bash"
	metadata.Mode = "legacy"

	rep, _ := reporter.GetReporter(format)
	output, err := rep.Generate(issues, metadata)
	if err != nil {
		return err
	}
	fmt.Println(string(output))

	if report.Summary.Critical > 0 {
		return &exitError{code: 2}
	} else if report.Summary.Total > 0 {
		return &exitError{code: 1}
	}
	return nil
}

func runListAnalyzers(_ *cobra.Command, _ []string) error {
	analyzers := analyzer.DefaultRegistry.List()
	if len(analyzers) == 0 {
		fmt.Println("No analyzers registered.")
		return nil
	}

	fmt.Println("Available Analyzers:")
	fmt.Println("--------------------")
	for _, a := range analyzers {
		modes := make([]string, len(a.SupportedModes()))
		for i, m := range a.SupportedModes() {
			modes[i] = m.String()
		}
		fmt.Printf("  %s\n", a.Name())
		fmt.Printf("    Category: %s\n", a.Category())
		fmt.Printf("    Description: %s\n", a.Description())
		fmt.Printf("    Modes: %v\n", modes)
		fmt.Println()
	}
	return nil
}
