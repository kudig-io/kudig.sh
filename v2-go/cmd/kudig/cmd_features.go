package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kudig/kudig/pkg/analyzer"
	"github.com/kudig/kudig/pkg/autofix"
	"github.com/kudig/kudig/pkg/collector"
	"github.com/kudig/kudig/pkg/cost"
	"github.com/kudig/kudig/pkg/rca"
	"github.com/kudig/kudig/pkg/reporter"
	"github.com/kudig/kudig/pkg/scanner"
	"github.com/kudig/kudig/pkg/tui"
	"github.com/kudig/kudig/pkg/types"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start interactive TUI mode",
	Long: `Start kudig in interactive Terminal User Interface mode.

The TUI provides an intuitive menu-driven interface for:
- Running online diagnostics
- Analyzing offline data
- Viewing history
- Configuring options

Example:
  kudig tui`,
	RunE: runTUI,
}

var rcaCmd = &cobra.Command{
	Use:   "rca",
	Short: "Perform root cause analysis on diagnostic results",
	Long: `Analyze diagnostic issues and identify root causes.

The RCA engine correlates multiple symptoms to identify underlying
root causes and suggests remediation actions.

Examples:
  kudig online --rca
  kudig rca /tmp/diagnose_1702468800`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRCA,
}

var grafanaCmd = &cobra.Command{
	Use:   "grafana",
	Short: "Export Grafana dashboard JSON",
	Long: `Export a Grafana dashboard JSON for visualizing kudig metrics.

Examples:
  kudig grafana > kudig-dashboard.json
  kudig grafana --output dashboard.json`,
	RunE: runGrafana,
}

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Auto-fix detected issues",
	Long: `Automatically fix detected issues where safe to do so.

Examples:
  kudig fix --dry-run
  kudig fix --confirm
  kudig fix --type IMAGE_PULL`,
	RunE: runFix,
}

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Analyze Kubernetes resource costs",
	Long: `Analyze and estimate Kubernetes resource costs.

Examples:
  kudig cost
  kudig cost --cpu-price 0.03 --memory-price 0.005`,
	RunE: runCost,
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan container images for vulnerabilities",
	Long: `Scan container images for security vulnerabilities.

Examples:
  kudig scan nginx:latest
  kudig scan --all-images`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScan,
}

func runTUI(_ *cobra.Command, _ []string) error {
	return tui.RunTUI()
}

func runRCA(_ *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var issues []types.Issue

	if len(args) > 0 {
		diagnosePath := args[0]
		col, ok := collector.GetCollector(types.ModeOffline)
		if !ok {
			return fmt.Errorf("offline collector not available")
		}
		config := collector.NewOfflineConfig(diagnosePath)
		data, err := col.Collect(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to collect data: %w", err)
		}
		results, err := analyzer.DefaultRegistry.ExecuteAll(ctx, data)
		if err != nil {
			return fmt.Errorf("failed to run analyzers: %w", err)
		}
		issues = analyzer.CollectIssues(results)
	} else {
		col, ok := collector.GetCollector(types.ModeOnline)
		if !ok {
			return fmt.Errorf("online collector not available")
		}
		config := &collector.Config{
			Kubeconfig:     kubeconfig,
			Context:        kubeCtx,
			NodeName:       nodeName,
			Namespace:      namespace,
			TimeoutSeconds: 60,
		}
		data, err := col.Collect(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to collect data: %w", err)
		}
		results, err := analyzer.DefaultRegistry.ExecuteByMode(ctx, data, types.ModeOnline)
		if err != nil {
			return fmt.Errorf("failed to run analyzers: %w", err)
		}
		issues = analyzer.CollectIssues(results)
	}

	engine := rca.NewEngine()
	rootCauses := engine.Analyze(ctx, issues)

	fmt.Printf("诊断发现 %d 个问题\n\n", len(issues))
	fmt.Println(rca.FormatRootCauses(rootCauses))
	return nil
}

func runGrafana(_ *cobra.Command, _ []string) error {
	generator := reporter.NewGrafanaDashboardGenerator()
	dashboard, err := generator.GenerateDashboard()
	if err != nil {
		return fmt.Errorf("failed to generate dashboard: %w", err)
	}
	fmt.Println(string(dashboard))
	return nil
}

func runFix(_ *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var issues []types.Issue

	if len(args) > 0 {
		diagnosePath := args[0]
		col, ok := collector.GetCollector(types.ModeOffline)
		if !ok {
			return fmt.Errorf("offline collector not available")
		}
		config := collector.NewOfflineConfig(diagnosePath)
		data, err := col.Collect(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to collect data: %w", err)
		}
		results, err := analyzer.DefaultRegistry.ExecuteAll(ctx, data)
		if err != nil {
			return fmt.Errorf("failed to run analyzers: %w", err)
		}
		issues = analyzer.CollectIssues(results)
	} else {
		col, ok := collector.GetCollector(types.ModeOnline)
		if !ok {
			return fmt.Errorf("online collector not available")
		}
		config := &collector.Config{
			Kubeconfig:     kubeconfig,
			Context:        kubeCtx,
			NodeName:       nodeName,
			Namespace:      namespace,
			TimeoutSeconds: 60,
		}
		data, err := col.Collect(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to collect data: %w", err)
		}
		results, err := analyzer.DefaultRegistry.ExecuteAll(ctx, data)
		if err != nil {
			return fmt.Errorf("failed to run analyzers: %w", err)
		}
		issues = analyzer.CollectIssues(results)
	}

	engine := autofix.NewEngine(true)

	fixable := engine.GetFixableIssues(issues)
	if len(fixable) == 0 {
		fmt.Println("没有可自动修复的问题")
		return nil
	}

	fmt.Printf("发现 %d 个可自动修复的问题（共 %d 个）:\n\n", len(fixable), len(issues))
	for _, issue := range fixable {
		action, _ := engine.CanFix(issue)
		fmt.Printf("  [%s] %s\n", issue.ENName, issue.CNName)
		fmt.Printf("    修复操作: %s (风险: %s)\n", action.Description, action.Risk)
		fmt.Printf("    命令: %s\n\n", action.Command)
	}

	fmt.Println("以上为 dry-run 模式预览。使用 --confirm 执行实际修复。")
	fmt.Println(autofix.FormatResults(engine.FixAll(ctx, fixable)))
	return nil
}

func runCost(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	col, ok := collector.GetCollector(types.ModeOnline)
	if !ok {
		return fmt.Errorf("online collector not available, cost analysis requires cluster access")
	}

	config := &collector.Config{
		Kubeconfig:     kubeconfig,
		Context:        kubeCtx,
		NodeName:       nodeName,
		Namespace:      namespace,
		TimeoutSeconds: 60,
	}

	data, err := col.Collect(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to collect data: %w", err)
	}

	a := cost.NewCostAnalyzer()
	result, err := a.Analyze(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to analyze costs: %w", err)
	}

	fmt.Println(cost.FormatResult(result))
	return nil
}

func runScan(_ *cobra.Command, args []string) error {
	image := "nginx:latest"
	if len(args) > 0 {
		image = args[0]
	}

	s := scanner.NewImageScanner()

	if !s.IsAvailable() {
		return fmt.Errorf("scanner %q not found in PATH; install trivy first: https://trivy.dev", s.ScannerType)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := s.ScanImage(ctx, image)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	fmt.Println(scanner.FormatResult(result))
	return nil
}
