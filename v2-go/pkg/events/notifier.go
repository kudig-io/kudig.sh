package events

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NotificationChannel is an abstraction for a target that can receive
// formatted event notifications (e.g. Slack, webhook, log, email).
type NotificationChannel interface {
	// Name returns a human-readable identifier for the channel.
	Name() string
	// Send delivers a formatted event message to the channel.
	Send(event Event, formatted string) error
}

// RateLimiter implements a token-bucket rate limiter that controls how many
// notifications can be sent within a given time window.
type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewRateLimiter creates a RateLimiter that allows maxTokens events per
// second at the given refill rate.
func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow consumes one token if available and returns true, otherwise returns false.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.refillRate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	rl.lastRefill = now

	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}
	return false
}

// Notifier receives events from the Watcher and dispatches them to registered
// NotificationChannels, applying rate limiting and mute durations.
type Notifier struct {
	eventCh       <-chan Event
	rateLimiter   *RateLimiter
	channels      []NotificationChannel
	muteDuration  time.Duration
	mutedEvents   map[string]time.Time // key -> last notified time
	mu            sync.RWMutex
}

// NewNotifier creates a Notifier that reads from eventCh and dispatches
// notifications. muteDuration controls how long identical events are suppressed.
func NewNotifier(eventCh <-chan Event, rateLimiter *RateLimiter, muteDuration time.Duration) *Notifier {
	return &Notifier{
		eventCh:      eventCh,
		rateLimiter:  rateLimiter,
		channels:     make([]NotificationChannel, 0),
		muteDuration: muteDuration,
		mutedEvents:  make(map[string]time.Time),
	}
}

// AddChannel registers a NotificationChannel for event delivery.
func (n *Notifier) AddChannel(ch NotificationChannel) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.channels = append(n.channels, ch)
}

// Start begins the notification processing loop. It blocks until ctx is cancelled.
func (n *Notifier) Start(ctx context.Context) {
	// Periodically clean up expired mute entries.
	go n.cleanupMuted(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-n.eventCh:
			if !ok {
				return
			}
			if !n.shouldNotify(event) {
				continue
			}
			if !n.rateLimiter.Allow() {
				continue
			}
			formatted := n.formatEvent(event)
			n.dispatch(event, formatted)
		}
	}
}

// shouldNotify determines whether the event should be dispatched based on
// mute rules. Warning and Error events bypass muting if they have not been
// seen within the mute window.
func (n *Notifier) shouldNotify(event Event) bool {
	if n.muteDuration <= 0 {
		return true
	}

	key := n.eventMuteKey(event)

	n.mu.RLock()
	lastNotified, exists := n.mutedEvents[key]
	n.mu.RUnlock()

	if exists && time.Since(lastNotified) < n.muteDuration {
		return false
	}

	n.mu.Lock()
	n.mutedEvents[key] = time.Now()
	n.mu.Unlock()

	return true
}

// eventMuteKey produces a deterministic key for mute deduplication.
func (n *Notifier) eventMuteKey(event Event) string {
	return fmt.Sprintf("%s:%s:%s:%s", event.Type, event.ResourceType, event.Namespace, event.ResourceName)
}

// dispatch sends the formatted message to all registered channels.
func (n *Notifier) dispatch(event Event, formatted string) {
	n.mu.RLock()
	channels := make([]NotificationChannel, len(n.channels))
	copy(channels, n.channels)
	n.mu.RUnlock()

	for _, ch := range channels {
		if err := ch.Send(event, formatted); err != nil {
			// Best-effort delivery; log the failure.
			fmt.Printf("[events] notification channel %s failed: %v\n", ch.Name(), err)
		}
	}
}

// formatEvent produces a human-readable string representation of an event
// suitable for notification messages.
func (n *Notifier) formatEvent(event Event) string {
	return fmt.Sprintf("[%s] %s/%s %s — %s: %s (count: %d, cluster: %s)",
		event.Timestamp.Format(time.RFC3339),
		event.Namespace,
		event.ResourceName,
		event.ResourceType,
		event.Reason,
		event.Message,
		event.Count,
		event.Cluster,
	)
}

// cleanupMuted periodically removes expired mute entries to prevent memory leaks.
func (n *Notifier) cleanupMuted(ctx context.Context) {
	ticker := time.NewTicker(n.muteDuration * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.mu.Lock()
			for key, t := range n.mutedEvents {
				if time.Since(t) >= n.muteDuration {
					delete(n.mutedEvents, key)
				}
			}
			n.mu.Unlock()
		}
	}
}
