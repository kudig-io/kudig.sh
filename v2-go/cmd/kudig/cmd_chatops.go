package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kudig/kudig/pkg/chatops"
	"github.com/kudig/kudig/pkg/chatops/messaging/dingtalk"
	"github.com/kudig/kudig/pkg/chatops/messaging/feishu"
)

var (
	chatopsPrefix      string
	chatopsMentionName string
	dingtalkEnabled    bool
	dingtalkAppKey     string
	dingtalkAppSecret  string
	dingtalkWebhook    string
	feishuEnabled      bool
	feishuAppID        string
	feishuAppSecret    string
	feishuWebhook      string
)

var chatopsCmd = &cobra.Command{
	Use:   "chatops",
	Short: "Start the ChatOps service for IM-based Kubernetes operations",
	Long: `Start a ChatOps service that connects to DingTalk and/or Feishu for
IM-based Kubernetes cluster operations.

Supports commands like:
  cluster status          — Show cluster health
  pod list [namespace]    — List pods
  deployment scale NAME N — Scale a deployment
  node list               — List nodes

Examples:
  kudig chatops --dingtalk --dingtalk-key=xxx --dingtalk-secret=yyy
  kudig chatops --feishu --feishu-appid=xxx --feishu-secret=yyy
  kudig chatops --dingtalk --feishu --prefix kudig`,
	RunE: runChatOps,
}

func runChatOps(_ *cobra.Command, _ []string) error {
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

	// Create command handler and router
	handler := chatops.NewHandler(clientset)
	router := chatops.NewCommandRouter(handler, chatopsPrefix, chatopsMentionName)

	// Create communicator manager
	manager := chatops.NewManager()

	// Register DingTalk if enabled
	if dingtalkEnabled {
		dtPlugin := dingtalk.NewPlugin(dingtalkAppKey, dingtalkAppSecret, dingtalkWebhook)
		manager.Register(dtPlugin)
		fmt.Fprintln(os.Stderr, "DingTalk ChatOps registered")
	}

	// Register Feishu if enabled
	if feishuEnabled {
		fsPlugin := feishu.NewPlugin(feishuAppID, feishuAppSecret, feishuWebhook)
		manager.Register(fsPlugin)
		fmt.Fprintln(os.Stderr, "Feishu ChatOps registered")
	}

	// Register command router as message handler
	manager.SetHandler(router.HandleMessage)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nShutting down ChatOps...")
		cancel()
	}()

	fmt.Fprintf(os.Stderr, "Starting kudig ChatOps (prefix: %s, mention: @%s)\n",
		chatopsPrefix, chatopsMentionName)

	if err := manager.StartAll(ctx); err != nil {
		return fmt.Errorf("failed to start ChatOps: %w", err)
	}

	<-ctx.Done()
	manager.StopAll()
	return nil
}

func init() {
	chatopsCmd.Flags().StringVar(&chatopsPrefix, "prefix", "kudig", "Command prefix for ChatOps")
	chatopsCmd.Flags().StringVar(&chatopsMentionName, "mention", "Kudig", "Bot mention name")

	chatopsCmd.Flags().BoolVar(&dingtalkEnabled, "dingtalk", false, "Enable DingTalk ChatOps")
	chatopsCmd.Flags().StringVar(&dingtalkAppKey, "dingtalk-key", "", "DingTalk App Key")
	chatopsCmd.Flags().StringVar(&dingtalkAppSecret, "dingtalk-secret", "", "DingTalk App Secret")
	chatopsCmd.Flags().StringVar(&dingtalkWebhook, "dingtalk-webhook", "", "DingTalk Webhook URL")

	chatopsCmd.Flags().BoolVar(&feishuEnabled, "feishu", false, "Enable Feishu ChatOps")
	chatopsCmd.Flags().StringVar(&feishuAppID, "feishu-appid", "", "Feishu App ID")
	chatopsCmd.Flags().StringVar(&feishuAppSecret, "feishu-secret", "", "Feishu App Secret")
	chatopsCmd.Flags().StringVar(&feishuWebhook, "feishu-webhook", "", "Feishu Webhook URL")

	rootCmd.AddCommand(chatopsCmd)
}
