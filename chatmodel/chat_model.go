package chatmodel

import (
	"context"
	"errors"
	"reAct-agent/agent"
	"reAct-agent/schema"
	"reAct-agent/tool"
	"strings"
	"time"
)

type ChatModelClient interface {
	Generate(ctx context.Context, model string, messages []*schema.Message, tools []*tool.ToolInfo) (*schema.Message, error)
	Stream(ctx context.Context, model string, messages []*schema.Message, tools []*tool.ToolInfo) (<-chan *schema.Message, <-chan error)
}

type ChatModelConfig struct {
	Client ChatModelClient

	APIKey  string
	Model   string
	BaseUrl string
	Timeout time.Duration
}

var _ agent.ChatModel = (*ChatModel)(nil)

// ChatModel represents a simple chat model that can generate responses
// and bind tool metadata for potential tool usage.
type ChatModel struct {
	conf   *ChatModelConfig
	client ChatModelClient
	tools  []*tool.ToolInfo
}

type ChatModelOption func(*ChatModelConfig)

// NewChatModel constructs a ChatModel.
func NewChatModel(ctx context.Context, config *ChatModelConfig, opts ...ChatModelOption) (*ChatModel, error) {
	for _, opt := range opts {
		opt(config)
	}
	if config.Client == nil {
		return nil, errors.New("client is required")
	}
	if len(config.Model) == 0 {
		return nil, errors.New("model is required")
	}
	if len(config.APIKey) == 0 {
		return nil, errors.New("api key is required")
	}
	if config.Timeout == 0 {
		config.Timeout = time.Minute
	}
	if len(config.BaseUrl) > 0 {
		if !strings.HasSuffix(config.BaseUrl, "/") {
			config.BaseUrl += "/"
		}
	}

	mdl := &ChatModel{conf: config, client: config.Client}
	return mdl, nil
}

// BindTools registers tool infos with the model.
func (c *ChatModel) BindTools(ctx context.Context, infos []*tool.ToolInfo) error {
	c.tools = infos
	return nil
}

// Generate produces a basic assistant message. In real usage, this would
// consult model logic and tool metadata.
func (c *ChatModel) Generate(ctx context.Context, history []*schema.Message) (*schema.Message, error) {
	msg, err := c.client.Generate(ctx, c.conf.Model, history, c.tools)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (c *ChatModel) Stream(ctx context.Context, history []*schema.Message) (<-chan *schema.Message, <-chan error) {
	return c.client.Stream(ctx, c.conf.Model, history, c.tools)
}
