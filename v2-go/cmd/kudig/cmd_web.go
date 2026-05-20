package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kudig/kudig/pkg/web"
)

var (
	webPort      int
	webStaticDir string
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the Web UI management server",
	Long: `Start a full-featured Web UI server for Kubernetes cluster management.

Provides REST API endpoints for clusters, pods, nodes, deployments, services,
and events. Serves a React SPA frontend for browser-based cluster management.

Examples:
  kudig web
  kudig web --port 8080
  kudig web --kubeconfig ~/.kube/config --port 3000`,
	RunE: runWeb,
}

func runWeb(_ *cobra.Command, _ []string) error {
	// Build kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}
	configOverrides := &clientcmd.ConfigOverrides{}
	if kubeCtx != "" {
		configOverrides.CurrentContext = kubeCtx
	}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, configOverrides,
	)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	srv := web.NewServer(clientset)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stderr, "Starting kudig Web UI on :%d\n", webPort)
		errCh <- srv.Start(webPort)
	}()

	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "\nReceived signal %v, shutting down...\n", sig)
		return nil
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}
}

func init() {
	webCmd.Flags().IntVarP(&webPort, "port", "p", 3000, "Port for the Web UI server")
	webCmd.Flags().StringVar(&webStaticDir, "static-dir", "./web/dist", "Path to static frontend files")
	rootCmd.AddCommand(webCmd)
}
