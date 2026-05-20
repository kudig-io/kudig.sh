package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/kudig/kudig/pkg/ai"
	"github.com/kudig/kudig/pkg/analyzer"
	"github.com/kudig/kudig/pkg/collector"
	"github.com/kudig/kudig/pkg/types"
)

var pprofCmd = &cobra.Command{
	Use:   "pprof",
	Short: "Run performance profiling on kudig",
	Long: `Start pprof profiling server for performance analysis.

Examples:
  kudig pprof
  kudig pprof --port 6060`,
	RunE: runPprof,
}

var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Start distributed tracing server",
	Long: `Start an OpenTelemetry tracing server for diagnostic tracing.

Examples:
  kudig trace
  kudig trace --jaeger http://localhost:14268`,
	RunE: runTrace,
}

var multiclusterCmd = &cobra.Command{
	Use:     "multicluster",
	Short:   "Diagnose multiple Kubernetes clusters",
	Aliases: []string{"mc"},
	Long: `Diagnose multiple Kubernetes clusters simultaneously.

Examples:
  kudig multicluster --all-contexts
  kudig multicluster --contexts prod-cluster,dr-cluster`,
	RunE: runMulticluster,
}

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-assisted diagnostic analysis",
	Long: `Use AI/LLM to analyze diagnostic results and provide insights.

Requires one of: OpenAI API key, Qwen API key, or Ollama running locally.

Examples:
  kudig ai /tmp/diagnose_1702468800
  kudig ai --online`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAI,
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for kudig.

To load completions:

Bash:
  $ source <(kudig completion bash)

Zsh:
  $ source <(kudig completion zsh)

Fish:
  $ kudig completion fish | source

PowerShell:
  PS> kudig completion powershell | Out-String | Invoke-Expression`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactArgs(1),
	RunE:                  runCompletion,
}

func runPprof(_ *cobra.Command, _ []string) error {
	addr := fmt.Sprintf("localhost:%d", pprofPort)
	fmt.Printf("Starting pprof server on http://%s/debug/pprof/\n", addr)
	fmt.Println("\nAvailable endpoints:")
	fmt.Printf("  http://%s/debug/pprof/           - Index\n", addr)
	fmt.Printf("  http://%s/debug/pprof/profile    - CPU Profile\n", addr)
	fmt.Printf("  http://%s/debug/pprof/heap       - Heap Profile\n", addr)
	fmt.Printf("  http://%s/debug/pprof/goroutine  - Goroutine Profile\n", addr)
	fmt.Printf("  http://%s/debug/pprof/allocs     - Allocations\n", addr)
	fmt.Println("\nPress Ctrl+C to stop")

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return http.ListenAndServe(addr, mux)
}

func runTrace(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("trace 功能尚未实现 (experimental): OpenTelemetry 集成正在开发中")
}

func runMulticluster(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("multicluster 功能尚未实现 (experimental): 多集群诊断正在开发中")
}

func runAI(_ *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	aiConfig := ai.LoadConfig()
	if aiConfig.APIKey == "" {
		return fmt.Errorf("AI 功能需要配置 API Key。请设置环境变量 KUDIG_AI_API_KEY，或使用 Ollama 本地模式")
	}

	assistant, err := ai.NewAssistant(aiConfig)
	if err != nil {
		return fmt.Errorf("failed to create AI assistant: %w", err)
	}

	if !assistant.IsAvailable() {
		return fmt.Errorf("AI provider not available (provider: %s)", aiConfig.Provider)
	}

	var issues []types.Issue

	if aiOnline {
		col, ok := collector.GetCollector(types.ModeOnline)
		if !ok {
			return fmt.Errorf("online collector not available")
		}
		colConfig := &collector.Config{
			Kubeconfig:     kubeconfig,
			Context:        kubeCtx,
			NodeName:       nodeName,
			Namespace:      namespace,
			TimeoutSeconds: 60,
		}
		data, err := col.Collect(ctx, colConfig)
		if err != nil {
			return fmt.Errorf("failed to collect data: %w", err)
		}
		results, err := analyzer.DefaultRegistry.ExecuteAll(ctx, data)
		if err != nil {
			return fmt.Errorf("failed to run analyzers: %w", err)
		}
		issues = analyzer.CollectIssues(results)
	} else if len(args) > 0 {
		col, ok := collector.GetCollector(types.ModeOffline)
		if !ok {
			return fmt.Errorf("offline collector not available")
		}
		colConfig := collector.NewOfflineConfig(args[0])
		data, err := col.Collect(ctx, colConfig)
		if err != nil {
			return fmt.Errorf("failed to collect data: %w", err)
		}
		results, err := analyzer.DefaultRegistry.ExecuteAll(ctx, data)
		if err != nil {
			return fmt.Errorf("failed to run analyzers: %w", err)
		}
		issues = analyzer.CollectIssues(results)
	} else {
		return fmt.Errorf("请指定离线诊断数据路径，或使用 --online 进行在线分析")
	}

	fmt.Printf("发现 %d 个问题，正在使用 AI 分析...\n\n", len(issues))

	result, err := assistant.AnalyzeWithAI(ctx, issues, "")
	if err != nil {
		return fmt.Errorf("AI analysis failed: %w", err)
	}

	fmt.Printf("=== AI 分析结果 ===\n\n")
	fmt.Printf("摘要: %s\n\n", result.Summary)
	if result.RootCause != "" {
		fmt.Printf("根因分析:\n%s\n\n", result.RootCause)
	}
	if len(result.Suggestions) > 0 {
		fmt.Println("修复建议:")
		for i, s := range result.Suggestions {
			fmt.Printf("  %d. [%s] %s\n", i+1, s.Risk, s.Description)
			if s.Command != "" {
				fmt.Printf("     命令: %s\n", s.Command)
			}
		}
	}
	fmt.Printf("\n置信度: %.0f%%\n", result.Confidence*100)
	return nil
}

func runCompletion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell type: %s. Use bash, zsh, fish, or powershell", args[0])
	}
}
