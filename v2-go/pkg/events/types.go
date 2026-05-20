// Package events provides Kubernetes event watching, filtering, and notification
// capabilities for the kudig monitoring system.
package events

import (
	"time"
)

// EventType represents the type of a Kubernetes event.
type EventType string

const (
	EventTypeNormal  EventType = "Normal"
	EventTypeWarning EventType = "Warning"
	EventTypeError   EventType = "Error"
	EventTypeCreated EventType = "Created"
	EventTypeUpdated EventType = "Updated"
	EventTypeDeleted EventType = "Deleted"
)

// ResourceType represents the kind of Kubernetes resource involved in an event.
type ResourceType string

const (
	ResourceTypePod       ResourceType = "Pod"
	ResourceTypeDeployment ResourceType = "Deployment"
	ResourceTypeService    ResourceType = "Service"
	ResourceTypeNode       ResourceType = "Node"
	ResourceTypeConfigMap  ResourceType = "ConfigMap"
	ResourceTypeSecret     ResourceType = "Secret"
	ResourceTypeIngress    ResourceType = "Ingress"
	ResourceTypePV         ResourceType = "PersistentVolume"
	ResourceTypePVC        ResourceType = "PersistentVolumeClaim"
)

// Severity represents the severity level of an event for notification purposes.
type Severity int

const (
	SeverityInfo     Severity = 0
	SeverityWarning  Severity = 1
	SeverityError    Severity = 2
	SeverityCritical Severity = 3
)

// String returns the string representation of a Severity.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// InvolvedObject references the Kubernetes object that the event is about.
type InvolvedObject struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	UID       string `json:"uid,omitempty"`
}

// Event represents a normalized Kubernetes event enriched with metadata
// for monitoring and notification purposes.
type Event struct {
	ID             string            `json:"id"`
	Type           EventType         `json:"type"`
	ResourceType   ResourceType      `json:"resourceType"`
	ResourceName   string            `json:"resourceName"`
	Namespace      string            `json:"namespace,omitempty"`
	Cluster        string            `json:"cluster,omitempty"`
	Reason         string            `json:"reason"`
	Message        string            `json:"message"`
	Timestamp      time.Time         `json:"timestamp"`
	Count          int32             `json:"count"`
	InvolvedObject InvolvedObject    `json:"involvedObject"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
}
