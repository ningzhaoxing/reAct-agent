package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"reAct-agent/schema"
	"reAct-agent/tool"
	"strings"
)

type ChatModel interface {
	Generate(ctx context.Context, history []*schema.Message) (*schema.Message, error)
	Stream(ctx context.Context, history []*schema.Message) (<-chan *schema.Message, <-chan error)
	BindTools(ctx context.Context, infos []*tool.ToolInfo) error
}

type MessageModifer func(ctx context.Context, msg []*schema.Message) []*schema.Message

type ReactAgentConfig struct {
	MaxStep int
	Model   ChatModel
	Tools   []tool.Tool
	// MessageModifier MessageModifer
}

// State tracks the conversation history.
type State struct {
	messages []*schema.Message
}

// ReactAgent wires ChatModel and Tool implementations per the UML diagram.
type ReactAgent struct {
	state *State
	conf  *ReactAgentConfig
}

type ReactAgentOption func(ra *ReactAgent)

func WithMaxStep(maxStep int) ReactAgentOption {
	return func(ra *ReactAgent) {
		ra.conf.MaxStep = maxStep
	}
}

// NewReactAgent constructs an agent with a model and tools, binding tool infos.
func NewReactAgent(ctx context.Context, conf *ReactAgentConfig, opts ...ReactAgentOption) (*ReactAgent, error) {
	ra := &ReactAgent{state: &State{messages: make([]*schema.Message, 0)}, conf: conf}
	for _, opt := range opts {
		opt(ra)
	}
	var infos []*tool.ToolInfo
	for _, t := range conf.Tools {
		info := t.Info()
		i := info
		infos = append(infos, &i)
	}
	if ra.conf.Model != nil {
		ra.conf.Model.BindTools(ctx, infos)
	}
	if ra.conf.MaxStep == 0 {
		ra.conf.MaxStep = 8
	}
	return ra, nil
}

// Generate delegates to the underlying ChatModel.
func (r *ReactAgent) Generate(ctx context.Context, history []*schema.Message) (*schema.Message, error, *State) {
	if r.conf.Model == nil {
		return &schema.Message{Role: schema.RoleAssistant, Content: "model not initialized"}, nil, nil
	}
	// 将用户输入加入 State
	r.state.messages = append(r.state.messages, history...)

	for step := 0; step < r.conf.MaxStep; step++ {
		// 交给 chatmodel 生成下一条消息
		msg, err := r.conf.Model.Generate(ctx, r.state.messages)
		if err != nil {
			return &schema.Message{Role: schema.RoleAssistant, Content: err.Error()}, err, nil
		}
		if msg == nil {
			return &schema.Message{Role: schema.RoleAssistant, Content: "empty message returned"}, nil, nil
		}

		// 如果是工具调用请求（role 为 Tool），执行工具
		if msg.Role == schema.RoleTool {
			// 记录模型的工具调用请求
			r.state.messages = append(r.state.messages, msg)

			// 从内容解析工具名与参数
			call, ok := parseToolCall(msg.Content)
			if !ok || call.Name == "" {
				return &schema.Message{Role: schema.RoleAssistant, Content: "invalid tool call payload"}, nil, nil
			}

			// 匹配工具
			var selected tool.Tool
			for _, t := range r.conf.Tools {
				if t.Info().Name == call.Name {
					selected = t
					break
				}
			}
			if selected == nil {
				return &schema.Message{Role: schema.RoleAssistant, Content: fmt.Sprintf("tool '%s' not found", call.Name)}, nil, nil
			}

			// 执行工具
			result, execErr := selected.Execute(ctx, call.Args)
			var toolContent string
			if execErr != nil {
				toolContent = fmt.Sprintf("{\"error\":\"%s\"}", escapeString(execErr.Error()))
			} else {
				if b, mErr := json.Marshal(result); mErr == nil {
					toolContent = string(b)
				} else {
					toolContent = fmt.Sprintf("{\"result\":\"%v\"}", result)
				}
			}

			// 将工具结果加入 State（role 仍为 Tool，内容为结果）
			r.state.messages = append(r.state.messages, &schema.Message{Role: schema.RoleTool, Content: toolContent})

			// 继续循环，让 chatmodel 根据工具结果决定下一步
			continue
		}

		// 如果是 assistant，退出循环并返回
		if msg.Role == schema.RoleAssistant {
			r.state.messages = append(r.state.messages, msg)
			return msg, nil, nil
		}

		// 其他角色（如 user/system），加入 State 并继续
		r.state.messages = append(r.state.messages, msg)
	}

	return &schema.Message{Role: schema.RoleAssistant, Content: "max steps reached"}, nil, r.state
}

// parseToolCall attempts to extract a tool invocation from assistant content.
// Supports JSON format: {"tool":"name","arguments":{...}}
// and ReAct text format: lines with "Action:" and "Action Input:".
func parseToolCall(content string) (struct {
	Name string
	Args map[string]interface{}
}, bool) {
	// 统一解析 JSON 格式，兼容多种字段命名
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(content), &raw); err == nil {
		// 尝试多种字段：tool/name/function.name
		var name string
		if v, ok := raw["tool"].(string); ok {
			name = v
		}
		if v, ok := raw["name"].(string); ok && name == "" {
			name = v
		}
		if fn, ok := raw["function"].(map[string]interface{}); ok && name == "" {
			if v, ok := fn["name"].(string); ok {
				name = v
			}
		}

		// 参数字段：arguments/args/input
		var args map[string]interface{}
		if v, ok := raw["arguments"].(map[string]interface{}); ok {
			args = v
		} else if v, ok := raw["args"].(map[string]interface{}); ok {
			args = v
		} else if v, ok := raw["input"].(map[string]interface{}); ok {
			args = v
		}

		if name != "" {
			return struct {
				Name string
				Args map[string]interface{}
			}{Name: name, Args: args}, true
		}
	}
	return struct {
		Name string
		Args map[string]interface{}
	}{}, false
}

func escapeString(s string) string {
	// minimal JSON string escape for quotes and newlines
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
