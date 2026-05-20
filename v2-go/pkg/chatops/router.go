package chatops

import (
	"context"
	"fmt"
	"strings"
)

// CommandRouter routes incoming IM messages to the appropriate command handler.
type CommandRouter struct {
	handler     *Handler
	prefix      string
	mentionName string
}

// NewCommandRouter creates a new CommandRouter.
func NewCommandRouter(handler *Handler, prefix, mentionName string) *CommandRouter {
	return &CommandRouter{
		handler:     handler,
		prefix:      prefix,
		mentionName: mentionName,
	}
}

// HandleMessage processes an incoming message, extracts the command, executes it,
// and returns a Response.
func (r *CommandRouter) HandleMessage(ctx context.Context, msg Message) (*Response, error) {
	command, ok := r.extractCommand(msg)
	if !ok {
		return nil, nil // not a command message, ignore
	}

	result, err := r.handler.HandleCommand(ctx, command)
	if err != nil {
		return &Response{
			Content: fmt.Sprintf("Error: %s", err.Error()),
			Format:  FormatMarkdown,
		}, nil
	}

	return &Response{
		Content: result,
		Format:  FormatMarkdown,
	}, nil
}

// extractCommand checks if the message is a chatops command and extracts
// the command string. Returns the command and true if it is a command,
// or empty string and false otherwise.
func (r *CommandRouter) extractCommand(msg Message) (string, bool) {
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return "", false
	}

	// Group messages must mention the bot (direct messages always count)
	if msg.IsGroup && !msg.Mentioned {
		return "", false
	}

	// Strip @mention from group messages
	if msg.IsGroup && r.mentionName != "" {
		content = strings.ReplaceAll(content, "@"+r.mentionName, "")
		content = strings.TrimSpace(content)
	}

	// If a prefix is configured, strip it
	if r.prefix != "" {
		if !strings.HasPrefix(content, r.prefix) {
			return "", false
		}
		content = strings.TrimPrefix(content, r.prefix)
		content = strings.TrimSpace(content)
	}

	return content, true
}
