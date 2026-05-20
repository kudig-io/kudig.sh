package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kudig/kudig/pkg/backup"
)

var (
	backupDir    string
	backupName   string
	backupFile   string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore Kubernetes cluster resources",
	Long: `Export cluster resources to JSON files for backup, or restore from a backup.

Supports 11 resource types: Namespaces, ServiceAccounts, Pods, Deployments,
Services, ConfigMaps, Secrets, Ingresses, NetworkPolicy, PVs, PVCs.

Examples:
  kudig backup create --name my-backup
  kudig backup list
  kudig backup restore --file backup-20260519.json
  kudig backup delete --file old-backup.json`,
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new cluster backup",
	RunE: func(_ *cobra.Command, _ []string) error {
		clientset, err := buildClientset()
		if err != nil {
			return err
		}
		mgr := backup.NewManager(clientset, backupDir)
		info, err := mgr.BackupCluster(context.Background(), backupName)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Backup created: %s (%d resources, %d bytes)\n",
			info.Name, info.ResourcesCount, info.Size)
		fmt.Println(info.FilePath)
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backups",
	RunE: func(_ *cobra.Command, _ []string) error {
		clientset, err := buildClientset()
		if err != nil {
			return err
		}
		mgr := backup.NewManager(clientset, backupDir)
		backups, err := mgr.ListBackups()
		if err != nil {
			return fmt.Errorf("list backups failed: %w", err)
		}
		if len(backups) == 0 {
			fmt.Println("No backups found.")
			return nil
		}
		fmt.Printf("%-30s %-20s %-10s %s\n", "NAME", "TIMESTAMP", "RESOURCES", "FILE")
		for _, b := range backups {
			fmt.Printf("%-30s %-20s %-10d %s\n", b.Name, b.Timestamp, b.ResourcesCount, filepath.Base(b.FilePath))
		}
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore cluster from a backup",
	RunE: func(_ *cobra.Command, _ []string) error {
		if backupFile == "" {
			return fmt.Errorf("--file is required")
		}
		clientset, err := buildClientset()
		if err != nil {
			return err
		}
		mgr := backup.NewManager(clientset, backupDir)
		if err := mgr.RestoreCluster(context.Background(), backupFile); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Backup restored from: %s\n", backupFile)
		return nil
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a backup",
	RunE: func(_ *cobra.Command, _ []string) error {
		if backupFile == "" {
			return fmt.Errorf("--file is required")
		}
		clientset, err := buildClientset()
		if err != nil {
			return err
		}
		mgr := backup.NewManager(clientset, backupDir)
		if err := mgr.DeleteBackup(backupFile); err != nil {
			return fmt.Errorf("delete failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Backup deleted: %s\n", backupFile)
		return nil
	},
}

func buildClientset() (*kubernetes.Clientset, error) {
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
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	return clientset, nil
}

func init() {
	homeDir, _ := os.UserHomeDir()
	defaultBackupDir := filepath.Join(homeDir, ".kudig", "backups")

	backupCmd.PersistentFlags().StringVar(&backupDir, "dir", defaultBackupDir, "Backup directory")

	backupCreateCmd.Flags().StringVar(&backupName, "name", "", "Backup name (auto-generated if empty)")
	backupRestoreCmd.Flags().StringVar(&backupFile, "file", "", "Backup file to restore")
	backupDeleteCmd.Flags().StringVar(&backupFile, "file", "", "Backup file to delete")

	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	rootCmd.AddCommand(backupCmd)
}
