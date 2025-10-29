package chatmodel

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

type ChatModelConfig struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
	BaseUrl    string
	Timeout    time.Duration
}

// ChatModel represents a simple chat model that can generate responses
// and bind tool metadata for potential tool usage.
type ChatModel struct {
	conf *ChatModelConfig

	tools []*tool.ToolInfo
}

// NewChatModel constructs a ChatModel.
func NewChatModel(ctx context.Context, config *ChatModelConfig) (*ChatModel, error) {
	if len(config.Model) == 0 {
		return nil, errors.New("model is required")
	}
	if len(config.APIKey) == 0 {
		return nil, errors.New("api key is required")
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}
	if config.Timeout == 0 {
		config.Timeout = time.Minute
	}
	if len(config.BaseUrl) > 0 {
		if !strings.HasSuffix(config.BaseUrl, "/") {
			config.BaseUrl += "/"
		}
	}

	return &ChatModel{conf: config}, nil
}

// BindTools registers tool infos with the model.
func (c *ChatModel) BindTools(ctx context.Context, infos []*tool.ToolInfo) error {
	c.tools = infos
	return nil
}

// Generate produces a basic assistant message. In real usage, this would
// consult model logic and tool metadata.
func (c *ChatModel) Generate(ctx context.Context, history []*schema.Message) (*schema.Message, error) {
	// Minimal stub implementation: echo last user message or default reply.
	var content string
	if len(history) > 0 {
		last := history[len(history)-1]
		if last != nil {
			content = "Received: " + last.Content
		}
	}
	return &schema.Message{Role: schema.RoleAssistant, Content: content}, nil
}
