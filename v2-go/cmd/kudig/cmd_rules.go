package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kudig/kudig/pkg/collector"
	"github.com/kudig/kudig/pkg/metrics"
	"github.com/kudig/kudig/pkg/reporter"
	"github.com/kudig/kudig/pkg/rules"
	"github.com/kudig/kudig/pkg/types"
)

var rulesCmd = &cobra.Command{
	Use:   "rules <diagnose_path>",
	Short: "Run custom YAML rules against diagnostic data",
	Long: `Run custom diagnostic rules defined in YAML files.

Examples:
  kudig rules /tmp/diagnose_1702468800
  kudig rules --file rules/custom.yaml /tmp/diagnose_1702468800
  kudig rules --dir rules/ /tmp/diagnose_1702468800
  kudig rules --list`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRules,
}

func runRules(_ *cobra.Command, args []string) error {
	loader := rules.NewLoader()

	if err := loader.LoadBuiltin(); err != nil {
		return fmt.Errorf("failed to load built-in rules: %w", err)
	}

	if rulesFile != "" {
		if err := loader.LoadFile(rulesFile); err != nil {
			return fmt.Errorf("failed to load rules file: %w", err)
		}
	}
	if rulesDir != "" {
		if err := loader.LoadDir(rulesDir); err != nil {
			return fmt.Errorf("failed to load rules directory: %w", err)
		}
	}

	if listRules {
		allRules := loader.GetAllRules()
		fmt.Println("Available Rules:")
		fmt.Println("----------------")
		for _, r := range allRules {
			fmt.Printf("  %s\n", r.ID)
			fmt.Printf("    Name: %s\n", r.Name)
			fmt.Printf("    Category: %s\n", r.Category)
			fmt.Printf("    Severity: %s\n", r.Severity)
			fmt.Printf("    Description: %s\n", r.Description)
			fmt.Println()
		}
		return nil
	}

	if len(args) < 1 {
		return fmt.Errorf("diagnose_path is required for rules analysis")
	}
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
		fmt.Fprintf(os.Stderr, "  kudig v%s - Rules Engine\n", version)
		fmt.Fprintf(os.Stderr, "================================================================\n\n")
		fmt.Fprintf(os.Stderr, "诊断目录: %s\n", diagnosePath)
		fmt.Fprintf(os.Stderr, "规则数量: %d\n", len(loader.GetAllRules()))
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

	engine := rules.NewEngine(loader)
	issues, err := engine.Evaluate(ctx, data)
	if err != nil {
		status = "rules_error"
		return fmt.Errorf("failed to evaluate rules: %w", err)
	}

	metrics.RecordIssues(issues)
	issues = reporter.DeduplicateIssues(issues)
	issues = reporter.SortIssuesBySeverity(issues)

	metadata := reporter.NewReportMetadata()
	metadata.Hostname = data.NodeInfo.Hostname
	metadata.DiagnosePath = diagnosePath
	metadata.Mode = "rules"
	metadata.Engine = "rules"

	rep, ok := reporter.GetReporter(format)
	if !ok {
		return fmt.Errorf("unknown format: %s", format)
	}

	output, err := rep.Generate(issues, metadata)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	if err := writeOutput(output); err != nil {
		return err
	}

	return severityExitCode(issues, &status)
}
