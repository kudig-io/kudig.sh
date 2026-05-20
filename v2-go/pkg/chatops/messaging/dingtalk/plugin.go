package dingtalk

import (
	"context"
	"fmt"
	"sync"

	"github.com/kudig/kudig/pkg/chatops"
)

// Plugin implements the chatops.Communicator interface for DingTalk.
type Plugin struct {
	client  *Client
	handler chatops.MessageHandler
	mu      sync.RWMutex
	done    chan struct{}
}

// NewPlugin creates a new DingTalk communicator plugin.
func NewPlugin(appKey, appSecret, webhook string) *Plugin {
	return &Plugin{
		client: NewClient(appKey, appSecret, webhook),
		done:   make(chan struct{}),
	}
}

// Name returns the platform name.
func (p *Plugin) Name() string {
	return "dingtalk"
}

// Start begins listening for DingTalk messages via the registered handler.
// Note: DingTalk outgoing webhook requires an HTTP server to receive callbacks.
// This is a placeholder for the callback server lifecycle.
func (p *Plugin) Start(ctx context.Context, handler chatops.MessageHandler) error {
	p.mu.Lock()
	p.handler = handler
	p.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
		case <-p.done:
		}
	}()

	return nil
}

// Stop gracefully stops the plugin.
func (p *Plugin) Stop() error {
	close(p.done)
	return nil
}

// SendMessage sends a chatops Response to the DingTalk webhook.
func (p *Plugin) SendMessage(ctx context.Context, channelID string, resp *chatops.Response) error {
	title := "kudig"
	if resp.Format == chatops.FormatMarkdown {
		title = "kudig - Kubernetes Info"
	}
	return p.client.SendMessage(title, resp.Content)
}

// OnMessage registers a handler for incoming DingTalk messages.
func (p *Plugin) OnMessage(handler chatops.MessageHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handler = handler
}

// HandleWebhook processes an incoming DingTalk webhook callback and returns a response.
func (p *Plugin) HandleWebhook(ctx context.Context, msg chatops.Message) (*chatops.Response, error) {
	p.mu.RLock()
	handler := p.handler
	p.mu.RUnlock()

	if handler == nil {
		return nil, fmt.Errorf("no message handler registered")
	}

	return handler(ctx, msg)
}
