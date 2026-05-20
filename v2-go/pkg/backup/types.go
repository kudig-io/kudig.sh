// Package backup provides Kubernetes cluster resource backup and restore
// capabilities, exporting selected resource types to JSON files on disk.
package backup

import "encoding/json"

// BackupInfo summarises a completed backup on disk.
type BackupInfo struct {
	Name            string `json:"name"`
	Timestamp       string `json:"timestamp"`
	FilePath        string `json:"filePath"`
	Size            int64  `json:"size"`
	ResourcesCount  int    `json:"resourcesCount"`
}

// BackupData is the on-disk representation of a full cluster backup.
type BackupData struct {
	Metadata  BackupMetadata               `json:"metadata"`
	Resources map[string][]json.RawMessage `json:"resources"`
}

// BackupMetadata records contextual information about a backup.
type BackupMetadata struct {
	Name        string `json:"name"`
	Timestamp   string `json:"timestamp"`
	ClusterName string `json:"clusterName"`
}
