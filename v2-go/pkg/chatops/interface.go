// Package chatops provides ChatOps integration for Kubernetes cluster management
// through instant messaging platforms.
package chatops

import (
	"context"
	"sync"
)

// FormatType defines the output format for chat responses.
type FormatType string

const (
	FormatPlain   FormatType = "plain"
	FormatMarkdown FormatType = "markdown"
	FormatJSON    FormatType = "json"
	FormatTable   FormatType = "table"
	FormatImage   FormatType = "image"
)

// Message represents an incoming IM message.
type Message struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	SenderID    string `json:"sender_id"`
	SenderName  string `json:"sender_name"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Timestamp   int64  `json:"timestamp"`
	IsGroup     bool   `json:"is_group"`
	Mentioned   bool   `json:"mentioned"`
}

// Response represents a chat response to be sent back.
type Response struct {
	Content     string                 `json:"content"`
	Format      FormatType             `json:"format"`
	Interactive *InteractiveComponent  `json:"interactive,omitempty"`
}

// InteractiveComponent contains interactive UI elements.
type InteractiveComponent struct {
	Type    string  `json:"type"`
	Buttons []Button `json:"buttons,omitempty"`
	Menus   []Menu   `json:"menus,omitempty"`
}

// Button represents an interactive button.
type Button struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Style string `json:"style"`
	Value string `json:"value"`
}

// Menu represents an interactive dropdown menu.
type Menu struct {
	ID      string       `json:"id"`
	Label   string       `json:"label"`
	Options []MenuOption `json:"options"`
}

// MenuOption represents an option in a dropdown menu.
type MenuOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// MessageHandler is a function that handles incoming messages and returns a response.
type MessageHandler func(ctx context.Context, msg Message) (*Response, error)

// Communicator defines the interface for IM platform integrations.
type Communicator interface {
	// Name returns the communicator platform name.
	Name() string

	// Start starts the communicator and begins listening for messages.
	Start(ctx context.Context, handler MessageHandler) error

	// Stop gracefully stops the communicator.
	Stop() error

	// SendMessage sends a response to the specified channel.
	SendMessage(ctx context.Context, channelID string, resp *Response) error

	// OnMessage registers a handler for incoming messages.
	OnMessage(handler MessageHandler)
}

// Manager manages registered communicators and routes messages.
type Manager struct {
	mu           sync.RWMutex
	communicators map[string]Communicator
	handler      MessageHandler
}

// NewManager creates a new chatops Manager.
func NewManager() *Manager {
	return &Manager{
		communicators: make(map[string]Communicator),
	}
}

// Register adds a communicator to the manager.
func (m *Manager) Register(c Communicator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.communicators[c.Name()] = c
}

// Get returns a registered communicator by name.
func (m *Manager) Get(name string) (Communicator, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.communicators[name]
	return c, ok
}

// SetHandler sets the global message handler for all communicators.
func (m *Manager) SetHandler(handler MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler
}

// StartAll starts all registered communicators.
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	handler := m.handler
	m.mu.RUnlock()

	for _, c := range m.communicators {
		if err := c.Start(ctx, handler); err != nil {
			return err
		}
	}
	return nil
}

// StopAll stops all registered communicators.
func (m *Manager) StopAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, c := range m.communicators {
		if err := c.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// List returns the names of all registered communicators.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.communicators))
	for name := range m.communicators {
		names = append(names, name)
	}
	return names
}
