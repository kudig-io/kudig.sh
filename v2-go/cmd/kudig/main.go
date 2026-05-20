// Package main is the entry point for kudig CLI
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "2.0.0"
	// Global flags
	verbose    bool
	outputFile string
	format     string

	// Online mode flags
	kubeconfig    string
	kubeCtx       string
	nodeName      string
	namespace     string
	allNodes      bool
	serveMetrics  bool
	metricsPort   int

	// Rules mode flags
	rulesFile string
	rulesDir  string
	listRules bool

	// RCA mode flags
	enableRCA bool

	// Pprof flags
	pprofPort int

	// Trace flags
	jaegerEndpoint string

	// Multicluster flags
	allContexts bool
	contexts    []string

	// AI flags
	aiOnline bool
)

// exitError 用于传递退出码而不直接调用 os.Exit。
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		if exitErr, ok := err.(*exitError); ok {
			os.Exit(exitErr.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "kudig",
	Short: "Kubernetes Diagnostic Toolkit",
	Long: `kudig (Kubernetes Diagnostic Toolkit) is a comprehensive diagnostic tool
for analyzing Kubernetes node issues.

It supports:
- Offline analysis of diagnostic data collected by diagnose_k8s.sh
- Online diagnosis via K8s API (real-time cluster analysis)
- Legacy mode using the original kudig.sh script`,
	Version: version,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Write output to file")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")

	// Online mode flags
	onlineCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	onlineCmd.Flags().StringVar(&kubeCtx, "context", "", "Kubernetes context to use")
	onlineCmd.Flags().StringVarP(&nodeName, "node", "n", "", "Node name to diagnose")
	onlineCmd.Flags().StringVar(&namespace, "namespace", "", "Namespace to focus on")
	onlineCmd.Flags().BoolVar(&allNodes, "all-nodes", false, "Diagnose all nodes")
	onlineCmd.Flags().BoolVar(&serveMetrics, "serve", false, "Start metrics server after diagnosis")
	onlineCmd.Flags().IntVar(&metricsPort, "metrics-port", 9090, "Port for metrics server")
	onlineCmd.Flags().BoolVar(&enableRCA, "rca", false, "Enable root cause analysis")

	// Rules mode flags
	rulesCmd.Flags().StringVar(&rulesFile, "file", "", "Path to rules YAML file")
	rulesCmd.Flags().StringVar(&rulesDir, "dir", "", "Path to rules directory")
	rulesCmd.Flags().BoolVar(&listRules, "list", false, "List available rules")

	// Add commands
	rootCmd.AddCommand(offlineCmd)
	rootCmd.AddCommand(onlineCmd)
	rootCmd.AddCommand(legacyCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(listAnalyzersCmd)
	rootCmd.AddCommand(rulesCmd)
	rootCmd.AddCommand(historyCmd)

	// Add history subcommands
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historyDiffCmd)

	// Add completion command
	rootCmd.AddCommand(completionCmd)

	// Add TUI command
	rootCmd.AddCommand(tuiCmd)

	// Add RCA command
	rootCmd.AddCommand(rcaCmd)

	// Add Grafana command
	rootCmd.AddCommand(grafanaCmd)

	// Add Fix command
	rootCmd.AddCommand(fixCmd)

	// Add Cost command
	rootCmd.AddCommand(costCmd)

	// Add Scan command
	rootCmd.AddCommand(scanCmd)

	// Add Pprof command
	rootCmd.AddCommand(pprofCmd)
	pprofCmd.Flags().IntVar(&pprofPort, "port", 6060, "Port for pprof server")

	// Add Trace command
	rootCmd.AddCommand(traceCmd)
	traceCmd.Flags().StringVar(&jaegerEndpoint, "jaeger", "", "Jaeger collector endpoint")

	// Add Multicluster command
	rootCmd.AddCommand(multiclusterCmd)
	multiclusterCmd.Flags().BoolVar(&allContexts, "all-contexts", false, "Diagnose all kubeconfig contexts")
	multiclusterCmd.Flags().StringSliceVar(&contexts, "contexts", []string{}, "Comma-separated list of contexts to diagnose")

	// Add AI command
	rootCmd.AddCommand(aiCmd)
	aiCmd.Flags().BoolVar(&aiOnline, "online", false, "Use online mode instead of offline path")

	// Add deprecated flags for backward compatibility
	rootCmd.Flags().Bool("json", false, "Output JSON format (deprecated, use --format json)")
}
