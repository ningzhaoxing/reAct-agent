package agent

import (
	"context"
	"reAct-agent/chatmodel"
	"reAct-agent/schema"
	"reAct-agent/tool"
)

type ReactAgentConfig struct {
	Model *chatmodel.ChatModel
	Tools []tool.Tool
}

// ReactAgent wires ChatModel and Tool implementations per the UML diagram.
type ReactAgent struct {
	conf *ReactAgentConfig
}

// NewReactAgent constructs an agent with a model and tools, binding tool infos.
func NewReactAgent(ctx context.Context, conf *ReactAgentConfig) *ReactAgent {
	ra := &ReactAgent{conf: conf}
	// Convert tools to ToolInfo and bind to model.
	var infos []*tool.ToolInfo
	for _, t := range conf.Tools {
		info := t.Info()
		// capture value into a new variable to take address safely
		i := info
		infos = append(infos, &i)
	}
	if ra.conf.Model != nil {
		ra.conf.Model.BindTools(ctx, infos)
	}
	return ra
}

// Generate delegates to the underlying ChatModel.
func (r *ReactAgent) Generate(ctx context.Context, history []*schema.Message) *schema.Message {
	if r.conf.Model == nil {
		return &schema.Message{Role: schema.RoleAssistant, Content: "model not initialized"}
	}
	msg, err := r.conf.Model.Generate(ctx, history)
	if err != nil {
		return &schema.Message{Role: schema.RoleAssistant, Content: err.Error()}
	}
	return msg
}
