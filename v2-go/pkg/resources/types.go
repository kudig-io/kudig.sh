// Package resources provides an extended Kubernetes resource manager
// supporting 27+ resource types across all major API groups.
package resources

import (
	"time"
)

// ResourceInfo represents a unified view of any Kubernetes resource.
type ResourceInfo struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	Kind              string            `json:"kind"`
	CreationTimestamp  time.Time         `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	Status            string            `json:"status,omitempty"`
	Raw               interface{}       `json:"raw,omitempty"`
}

// ResourceList holds a collection of ResourceInfo items for a given resource kind.
type ResourceList struct {
	Items        []ResourceInfo `json:"items"`
	Total        int            `json:"total"`
	ResourceKind string         `json:"resourceKind"`
}
