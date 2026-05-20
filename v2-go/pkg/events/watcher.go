package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// resourceWatch holds a cancel function for an active watch on a resource type.
type resourceWatch struct {
	cancel context.CancelFunc
}

// Watcher watches Kubernetes resources for events and emits normalized Event
// objects on a channel for downstream processing.
type Watcher struct {
	clientset kubernetes.Interface
	watchers  map[ResourceType]*resourceWatch
	mu        sync.RWMutex
	events    chan Event
	cluster   string
}

// NewWatcher creates a new event Watcher for the given Kubernetes clientset.
// The cluster name is attached to every emitted event for multi-cluster contexts.
func NewWatcher(clientset kubernetes.Interface, clusterName string) *Watcher {
	return &Watcher{
		clientset: clientset,
		watchers:  make(map[ResourceType]*resourceWatch),
		events:    make(chan Event, 256),
		cluster:   clusterName,
	}
}

// Events returns the read-only channel on which normalized events are delivered.
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Start begins watching Pods, Deployments, Services, and Nodes. It blocks
// until ctx is cancelled, at which point all watchers are torn down.
func (w *Watcher) Start(ctx context.Context) error {
	resources := []ResourceType{
		ResourceTypePod,
		ResourceTypeDeployment,
		ResourceTypeService,
		ResourceTypeNode,
	}

	for _, rt := range resources {
		if err := w.watchResource(ctx, rt); err != nil {
			// Log but continue — one resource failing should not block others.
			klog.ErrorS(err, "Failed to start watch", "resource", rt)
		}
	}

	// Block until context cancellation.
	<-ctx.Done()
	w.Stop()
	return nil
}

// Stop terminates all active watchers and closes the events channel.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for rt, rw := range w.watchers {
		rw.cancel()
		delete(w.watchers, rt)
	}
	close(w.events)
}

// watchResource starts a Kubernetes watch for the given resource type.
func (w *Watcher) watchResource(ctx context.Context, rt ResourceType) error {
	watchCtx, watchCancel := context.WithCancel(ctx)

	var watcher watch.Interface
	var err error

	switch rt {
	case ResourceTypePod:
		watcher, err = w.clientset.CoreV1().Pods("").Watch(watchCtx, metav1.ListOptions{})
	case ResourceTypeDeployment:
		watcher, err = w.clientset.AppsV1().Deployments("").Watch(watchCtx, metav1.ListOptions{})
	case ResourceTypeService:
		watcher, err = w.clientset.CoreV1().Services("").Watch(watchCtx, metav1.ListOptions{})
	case ResourceTypeNode:
		watcher, err = w.clientset.CoreV1().Nodes().Watch(watchCtx, metav1.ListOptions{})
	default:
		watchCancel()
		return fmt.Errorf("unsupported resource type: %s", rt)
	}

	if err != nil {
		watchCancel()
		return fmt.Errorf("failed to create watch for %s: %w", rt, err)
	}

	w.mu.Lock()
	w.watchers[rt] = &resourceWatch{cancel: watchCancel}
	w.mu.Unlock()

	go w.processWatchEvents(watchCtx, watcher, rt)

	return nil
}

// processWatchEvents reads events from the Kubernetes watcher, converts them,
// and sends them on the output channel.
func (w *Watcher) processWatchEvents(ctx context.Context, watcher watch.Interface, rt ResourceType) {
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-watcher.ResultChan():
			if !ok {
				klog.V(2).InfoS("Watch channel closed", "resource", rt)
				return
			}
			converted := w.convertEvent(evt, rt)
			if converted == nil {
				continue
			}
			select {
			case w.events <- *converted:
			case <-ctx.Done():
				return
			default:
				// Drop events if the buffer is full to avoid blocking.
				klog.V(4).InfoS("Event dropped due to full buffer", "resource", rt)
			}
		}
	}
}

// convertEvent translates a raw Kubernetes watch.Event into a normalized
// events.Event. Returns nil for event types we do not propagate.
func (w *Watcher) convertEvent(evt watch.Event, rt ResourceType) *Event {
	eventType := w.mapWatchType(evt.Type)
	if eventType == "" {
		return nil
	}

	var e Event
	e.Type = eventType
	e.ResourceType = rt
	e.Cluster = w.cluster
	e.Timestamp = time.Now().UTC()

	switch obj := evt.Object.(type) {
	case *corev1.Pod:
		e.ID = string(obj.UID)
		e.ResourceName = obj.Name
		e.Namespace = obj.Namespace
		e.Reason = string(evt.Type)
		e.Message = fmt.Sprintf("Pod %s/%s %s", obj.Namespace, obj.Name, evt.Type)
		e.InvolvedObject = InvolvedObject{
			Kind:      "Pod",
			Name:      obj.Name,
			Namespace: obj.Namespace,
			UID:       string(obj.UID),
		}
		e.Labels = obj.Labels
		e.Annotations = obj.Annotations

	case *corev1.Service:
		e.ID = string(obj.UID)
		e.ResourceName = obj.Name
		e.Namespace = obj.Namespace
		e.Reason = string(evt.Type)
		e.Message = fmt.Sprintf("Service %s/%s %s", obj.Namespace, obj.Name, evt.Type)
		e.InvolvedObject = InvolvedObject{
			Kind:      "Service",
			Name:      obj.Name,
			Namespace: obj.Namespace,
			UID:       string(obj.UID),
		}
		e.Labels = obj.Labels
		e.Annotations = obj.Annotations

	case *corev1.Node:
		e.ID = string(obj.UID)
		e.ResourceName = obj.Name
		e.Reason = string(evt.Type)
		e.Message = fmt.Sprintf("Node %s %s", obj.Name, evt.Type)
		e.InvolvedObject = InvolvedObject{
			Kind: "Node",
			Name: obj.Name,
			UID:  string(obj.UID),
		}
		e.Labels = obj.Labels
		e.Annotations = obj.Annotations

	default:
		// Handle Deployments via unstructured or runtime.Object fallback.
		type accessor interface {
			GetUID() string
			GetName() string
			GetNamespace() string
			GetLabels() map[string]string
			GetAnnotations() map[string]string
		}
		if a, ok := evt.Object.(accessor); ok {
			e.ID = string(a.GetUID())
			e.ResourceName = a.GetName()
			e.Namespace = a.GetNamespace()
			e.Reason = string(evt.Type)
			e.Message = fmt.Sprintf("%s %s/%s %s", rt, a.GetNamespace(), a.GetName(), evt.Type)
			e.InvolvedObject = InvolvedObject{
				Kind:      string(rt),
				Name:      a.GetName(),
				Namespace: a.GetNamespace(),
				UID:       string(a.GetUID()),
			}
			e.Labels = a.GetLabels()
			e.Annotations = a.GetAnnotations()
		} else {
			return nil
		}
	}

	// Derive count from annotations if present.
	if countStr, ok := e.Annotations["kudig.io/event-count"]; ok != false {
		var count int32
		fmt.Sscanf(countStr, "%d", &count)
		if count > 0 {
			e.Count = count
		}
	}
	if e.Count == 0 {
		e.Count = 1
	}

	return &e
}

// mapWatchType converts a Kubernetes watch.EventType to our EventType.
// Returns empty string for types we skip (e.g. Bookmark, Error).
func (w *Watcher) mapWatchType(t watch.EventType) EventType {
	switch t {
	case watch.Added:
		return EventTypeCreated
	case watch.Modified:
		return EventTypeUpdated
	case watch.Deleted:
		return EventTypeDeleted
	default:
		return ""
	}
}
