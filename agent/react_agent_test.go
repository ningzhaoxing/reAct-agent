package agent_test

import (
	"context"
	"reAct-agent/agent"
	"reAct-agent/chatmodel"
	"reAct-agent/schema"
	"reAct-agent/tool"
	"testing"
)

func TestNewReactAgent(t *testing.T) {
	ctx := context.Background()
	apiKey := "sk-00e87cc20b4f909f8c60a0635a69e074940ea275196e0615658384a3828f4b62"
	baseUrl := "https://openai.qiniu.com/v1"
	qwModel, err := chatmodel.NewQWenModelClient(apiKey, chatmodel.WithBaseUrl(baseUrl))
	if err != nil {
		t.Fatalf("NewQWenModelClient failed: %v", err)
	}
	chatModel, err := chatmodel.NewChatModel(ctx, &chatmodel.ChatModelConfig{
		Client: qwModel,
		APIKey: apiKey,
		Model:  "qwen3-coder-480b-a35b-instruct",
	})
	if err != nil {
		t.Fatalf("NewChatModel failed: %v", err)
	}
	reactAgent, err := agent.NewReactAgent(ctx, &agent.ReactAgentConfig{
		Model: chatModel,
		Tools: []tool.Tool{
			&tool.CalculatorTool{},
		},
	})

	res, err, state := reactAgent.Generate(ctx, []*schema.Message{
		{Role: schema.RoleUser, Content: "What is 2 + 2?"},
	})
	if err != nil {
		t.Fatalf("NewReactAgent failed: %v", err)
	}

	t.Log(res, state)
}
