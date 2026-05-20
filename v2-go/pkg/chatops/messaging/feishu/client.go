// Package feishu provides Feishu (Lark) webhook integration for ChatOps.
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/kudig/kudig/pkg/chatops"
)

// Client sends messages to Feishu via webhook.
type Client struct {
	appID     string
	appSecret string
	webhook   string
	httpClient *http.Client
}

// feishuRequest is the Feishu webhook card message payload.
type feishuRequest struct {
	MsgType string     `json:"msgtype"`
	Card    feishuCard `json:"card"`
}

type feishuCard struct {
	Header   feishuCardHeader `json:"header"`
	Elements []feishuElement  `json:"elements"`
}

type feishuCardHeader struct {
	Title    feishuText `json:"title"`
	Template string     `json:"template,omitempty"`
}

type feishuText struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

type feishuElement struct {
	Tag     string `json:"tag"`
	Content string `json:"content,omitempty"`
}

// feishuResponse is the Feishu webhook API response.
type feishuResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// NewClient creates a new Feishu webhook client.
func NewClient(appID, appSecret, webhook string) *Client {
	return &Client{
		appID:     appID,
		appSecret: appSecret,
		webhook:   webhook,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendMessage sends an interactive card message to Feishu webhook.
func (c *Client) SendMessage(ctx context.Context, title, markdown string) error {
	payload := feishuRequest{
		MsgType: "interactive",
		Card: feishuCard{
			Header: feishuCardHeader{
				Title: feishuText{
					Content: title,
					Tag:     "plain_text",
				},
				Template: "blue",
			},
			Elements: []feishuElement{
				{
					Tag:     "markdown",
					Content: markdown,
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhook, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result feishuResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("feishu error %d: %s", result.Code, result.Msg)
	}

	return nil
}

// Plugin implements the chatops.Communicator interface for Feishu.
type Plugin struct {
	client  *Client
	handler chatops.MessageHandler
	mu      sync.RWMutex
	done    chan struct{}
}

// NewPlugin creates a new Feishu communicator plugin.
func NewPlugin(appID, appSecret, webhook string) *Plugin {
	return &Plugin{
		client: NewClient(appID, appSecret, webhook),
		done:   make(chan struct{}),
	}
}

// Name returns the platform name.
func (p *Plugin) Name() string {
	return "feishu"
}

// Start begins listening for Feishu messages via the registered handler.
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

// SendMessage sends a chatops Response to the Feishu webhook.
func (p *Plugin) SendMessage(ctx context.Context, channelID string, resp *chatops.Response) error {
	title := "kudig - Kubernetes Info"
	return p.client.SendMessage(ctx, title, resp.Content)
}

// OnMessage registers a handler for incoming Feishu messages.
func (p *Plugin) OnMessage(handler chatops.MessageHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handler = handler
}
