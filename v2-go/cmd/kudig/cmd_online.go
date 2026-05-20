package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"

	"github.com/kudig/kudig/pkg/analyzer"
	"github.com/kudig/kudig/pkg/collector"
	"github.com/kudig/kudig/pkg/collector/online"
	"github.com/kudig/kudig/pkg/history"
	"github.com/kudig/kudig/pkg/metrics"
	"github.com/kudig/kudig/pkg/reporter"
	"github.com/kudig/kudig/pkg/types"
)

var onlineCmd = &cobra.Command{
	Use:   "online",
	Short: "Diagnose a live Kubernetes cluster",
	Long: `Perform real-time diagnosis of a Kubernetes cluster via K8s API.

Examples:
  kudig online
  kudig online --kubeconfig ~/.kube/config --node worker-1
  kudig online --all-nodes
  kudig online --namespace my-app`,
	RunE: runOnline,
}

func runOnline(_ *cobra.Command, _ []string) error {
	if serveMetrics {
		addr := fmt.Sprintf(":%d", metricsPort)
		metricsServer := metrics.NewServer(addr)
		go func() {
			if verbose {
				fmt.Fprintf(os.Stderr, "Starting metrics server on %s/metrics\n", addr)
			}
			if err := metricsServer.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Metrics server error: %v\n", err)
			}
		}()
	}

	startTime := time.Now()
	status := "success"
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordDiagnosis(types.ModeOnline, status, duration)
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
		fmt.Fprintf(os.Stderr, "  kudig v%s - Kubernetes Diagnostic Toolkit (Online Mode)\n", version)
		fmt.Fprintf(os.Stderr, "================================================================\n\n")
		fmt.Fprintf(os.Stderr, "分析时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		if nodeName != "" {
			fmt.Fprintf(os.Stderr, "目标节点: %s\n", nodeName)
		}
		if namespace != "" {
			fmt.Fprintf(os.Stderr, "目标命名空间: %s\n", namespace)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	col, ok := collector.GetCollector(types.ModeOnline)
	if !ok {
		status = "collector_error"
		return fmt.Errorf("online collector not available")
	}

	if allNodes {
		return runOnlineAllNodes(ctx, col, &status)
	}
	return runOnlineSingleNode(ctx, col, &status)
}

func runOnlineSingleNode(ctx context.Context, col collector.Collector, status *string) error {
	config := &collector.Config{
		Kubeconfig:     kubeconfig,
		Context:        kubeCtx,
		NodeName:       nodeName,
		Namespace:      namespace,
		AllNodes:       false,
		TimeoutSeconds: 60,
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "正在连接 Kubernetes 集群...\n")
	}

	data, err := col.Collect(ctx, config)
	if err != nil {
		*status = "collect_error"
		return fmt.Errorf("failed to collect data: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "已连接到集群\n")
		if data.NodeInfo.Hostname != "" {
			fmt.Fprintf(os.Stderr, "节点: %s\n", data.NodeInfo.Hostname)
			fmt.Fprintf(os.Stderr, "Kubelet版本: %s\n", data.NodeInfo.KubeletVersion)
			fmt.Fprintf(os.Stderr, "容器运行时: %s\n", data.NodeInfo.ContainerRuntime)
		}
		fmt.Fprintf(os.Stderr, "\n开始诊断检查...\n\n")
	}

	results, err := analyzer.DefaultRegistry.ExecuteByMode(ctx, data, types.ModeOnline)
	if err != nil {
		*status = "analyze_error"
		return fmt.Errorf("failed to run analyzers: %w", err)
	}

	metrics.RecordAnalyzers(types.ModeOnline, len(results))
	issues := analyzer.CollectIssues(results)
	metrics.RecordIssues(issues)

	issues = reporter.DeduplicateIssues(issues)
	issues = reporter.SortIssuesBySeverity(issues)

	metadata := reporter.NewReportMetadata()
	metadata.Hostname = data.NodeInfo.Hostname
	metadata.Mode = "online"
	if kubeconfig != "" {
		metadata.DiagnosePath = kubeconfig
	}

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

	if histMgr, err := history.NewManager(); err == nil {
		if _, err := histMgr.Save(data.NodeInfo.Hostname, "online", issues); err == nil && verbose {
			fmt.Fprintf(os.Stderr, "已保存到历史记录\n")
		}
	}

	sendNotification(data.NodeInfo.Hostname, "online", issues)
	return severityExitCode(issues, status)
}

func runOnlineAllNodes(ctx context.Context, col collector.Collector, status *string) error {
	config := &collector.Config{
		Kubeconfig:     kubeconfig,
		Context:        kubeCtx,
		NodeName:       "",
		Namespace:      namespace,
		AllNodes:       true,
		TimeoutSeconds: 60,
	}

	onlineCollector, ok := col.(*online.Collector)
	if !ok {
		*status = "collector_type_error"
		return fmt.Errorf("collector is not an online collector")
	}

	var bar *progressbar.ProgressBar
	if verbose {
		fmt.Fprintf(os.Stderr, "正在收集所有节点数据...\n")
	} else {
		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription("Diagnosing nodes"),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetItsString("nodes"),
			progressbar.OptionThrottle(100*time.Millisecond),
			progressbar.OptionShowElapsedTimeOnFinish(),
		)
	}

	progressFn := func(current, total int, nodeName string) {
		if bar != nil {
			bar.ChangeMax(total)
			bar.Set(current)
		} else if verbose {
			fmt.Fprintf(os.Stderr, "  [%d/%d] Diagnosing node: %s\n", current, total, nodeName)
		}
	}

	nodeResults, err := onlineCollector.CollectAllNodesConcurrent(ctx, config, progressFn)
	if err != nil {
		*status = "collect_error"
		return fmt.Errorf("failed to collect nodes data: %w", err)
	}

	if bar != nil {
		bar.Finish()
		fmt.Fprintln(os.Stderr)
	}

	allIssues := make([]types.Issue, 0)
	successfulNodes := 0
	failedNodes := 0

	for _, result := range nodeResults {
		if result.Error != nil {
			failedNodes++
			fmt.Fprintf(os.Stderr, "Warning: failed to diagnose node %s: %v\n", result.NodeName, result.Error)
			continue
		}

		successfulNodes++

		results, err := analyzer.DefaultRegistry.ExecuteByMode(ctx, result.Data, types.ModeOnline)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to run analyzers on node %s: %v\n", result.NodeName, err)
			continue
		}

		issues := analyzer.CollectIssues(results)
		for i := range issues {
			if issues[i].Metadata == nil {
				issues[i].Metadata = make(map[string]string)
			}
			issues[i].Metadata["node"] = result.NodeName
			issues[i].Details = fmt.Sprintf("[%s] %s", result.NodeName, issues[i].Details)
		}

		allIssues = append(allIssues, issues...)
		metrics.RecordAnalyzers(types.ModeOnline, len(results))
	}

	if successfulNodes == 0 {
		*status = "all_nodes_failed"
		return fmt.Errorf("failed to diagnose any node")
	}

	metrics.RecordIssues(allIssues)
	allIssues = reporter.DeduplicateIssues(allIssues)
	allIssues = reporter.SortIssuesBySeverity(allIssues)

	metadata := reporter.NewReportMetadata()
	metadata.Hostname = fmt.Sprintf("%d nodes", successfulNodes)
	metadata.Mode = "online-multi-node"
	if kubeconfig != "" {
		metadata.DiagnosePath = kubeconfig
	}

	rep, ok := reporter.GetReporter(format)
	if !ok {
		return fmt.Errorf("unknown format: %s", format)
	}

	output, err := rep.Generate(allIssues, metadata)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	if err := writeOutput(output); err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "\n诊断完成: %d 个节点成功, %d 个节点失败\n", successfulNodes, failedNodes)
	}

	if histMgr, err := history.NewManager(); err == nil {
		histLabel := fmt.Sprintf("%d nodes", successfulNodes)
		if _, err := histMgr.Save(histLabel, "online-multi-node", allIssues); err == nil && verbose {
			fmt.Fprintf(os.Stderr, "已保存到历史记录\n")
		}
	}

	sendNotification(fmt.Sprintf("%d nodes", successfulNodes), "online-multi-node", allIssues)
	return severityExitCode(allIssues, status)
}
