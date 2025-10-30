package chatmodel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	httpclient "reAct-agent/http_client"
	"reAct-agent/schema"
	"reAct-agent/tool"
	"strings"
	"time"
)

type QWenModelClient struct {
	BaseUrl   string
	AuthToken string
	Timeout   time.Duration
	Path      string // default: chat/completions

	HTTPClient httpclient.IHTTPClient
}

// QWenRequest represents the request structure for QWen API
type QWenRequest struct {
	Model    string                   `json:"model"`
	Messages []QWenMessage            `json:"messages"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
	Stream   bool                     `json:"stream,omitempty"`
}

// QWenMessage represents a message in QWen API format
type QWenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// QWenResponse represents the response structure for QWen API
type QWenResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []QWenChoice `json:"choices"`
	Usage   QWenUsage    `json:"usage"`
}

// QWenChoice represents a choice in the response
type QWenChoice struct {
	Index        int         `json:"index"`
	Message      QWenMessage `json:"message"`
	Delta        QWenMessage `json:"delta,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

// QWenUsage represents token usage information
type QWenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// QWenStreamResponse represents a streaming response chunk
type QWenStreamResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []QWenChoice `json:"choices"`
}

type Option func(*QWenModelClient) error

func WithBaseUrl(baseUrl string) Option {
	return func(c *QWenModelClient) error {
		c.BaseUrl = baseUrl
		return nil
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *QWenModelClient) error {
		c.Timeout = timeout
		return nil
	}
}

func WithHTTPClient(httpClient httpclient.IHTTPClient) Option {
	return func(c *QWenModelClient) error {
		c.HTTPClient = httpClient
		return nil
	}
}

func NewQWenModelClient(authToken string, opts ...Option) (*QWenModelClient, error) {
	if authToken == "" {
		return nil, errors.New("authToken is required")
	}

	client := &QWenModelClient{
		AuthToken: authToken,
		Timeout:   5 * time.Minute,
		Path:      "chat/completions",
	}

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	// Initialize default HTTP client if not provided
	if client.HTTPClient == nil {
		base := client.BaseUrl
		if base == "" {
			base = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
		header := httpclient.HTTPHeader{
			"Content-Type":  "application/json",
			"Accept":        "application/json",
			"Authorization": "Bearer " + client.AuthToken,
		}
		client.HTTPClient = httpclient.NewHTTPClient(base, client.Path,
			httpclient.WithHeader(header),
			httpclient.WithTimeout(client.Timeout),
		)
	}

	return client, nil
}

// GenerateMessage 调用 QWen API 获取完整响应
func (c *QWenModelClient) Generate(ctx context.Context, model string, messages []*schema.Message, tools []*tool.ToolInfo) (*schema.Message, error) {
	// 构建请求
	reqMessages := make([]QWenMessage, len(messages))
	for i, msg := range messages {
		reqMessages[i] = QWenMessage{
			Role:    msg.Role.String(),
			Content: msg.Content,
		}
	}

	qwenReq := QWenRequest{
		Model:    model,
		Messages: reqMessages,
		Stream:   false,
	}

	// 添加工具信息
	if len(tools) > 0 {
		qwenTools := make([]map[string]interface{}, len(tools))
		for i, toolInfo := range tools {
			qwenTools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        toolInfo.Name,
					"description": toolInfo.Desc,
					"parameters":  toolInfo.Parameters,
				},
			}
		}
		qwenReq.Tools = qwenTools
	}

	// 使用接口客户端发送请求
	httpResp, err := c.HTTPClient.Send(ctx, httpclient.HTTPMethodPOST, qwenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode, string(httpResp.Body))
	}

	// 解析响应
	var qwenResp QWenResponse
	if err := json.Unmarshal(httpResp.Body, &qwenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 检查是否有返回的选择
	if len(qwenResp.Choices) == 0 {
		return nil, errors.New("no choices returned from API")
	}

	// 转换为 schema.Message
	choice := qwenResp.Choices[0]
	return &schema.Message{
		Role:    schema.RoleAssistant,
		Content: choice.Message.Content,
	}, nil
}

// GenerateMessageStream 通过流式方式调用 QWen API
func (c *QWenModelClient) Stream(ctx context.Context, model string, messages []*schema.Message, tools []*tool.ToolInfo) (<-chan *schema.Message, <-chan error) {
	msgChan := make(chan *schema.Message, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(msgChan)
		defer close(errChan)

		// 构建请求
		reqMessages := make([]QWenMessage, len(messages))
		for i, msg := range messages {
			reqMessages[i] = QWenMessage{
				Role:    msg.Role.String(),
				Content: msg.Content,
			}
		}

		qwenReq := QWenRequest{
			Model:    model,
			Messages: reqMessages,
			Stream:   true,
		}

		// 添加工具信息（如果有）
		if len(tools) > 0 {
			qwenTools := make([]map[string]interface{}, len(tools))
			for i, toolInfo := range tools {
				qwenTools[i] = map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":        toolInfo.Name,
						"description": toolInfo.Desc,
						"parameters":  toolInfo.Parameters,
					},
				}
			}
			qwenReq.Tools = qwenTools
		}

		// 为流式创建 Accept 为 SSE 的客户端临时实例
		base := c.BaseUrl
		if base == "" {
			base = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
		header := httpclient.HTTPHeader{
			"Content-Type":  "application/json",
			"Accept":        "text/event-stream",
			"Authorization": "Bearer " + c.AuthToken,
		}
		sseClient := httpclient.NewHTTPClient(base, c.Path,
			httpclient.WithHeader(header),
			httpclient.WithTimeout(c.Timeout),
		)

		stream, errs := sseClient.SendStream(ctx, httpclient.HTTPMethodPOST, qwenReq)

		// 读取流式响应与解析 SSE
		var buf bytes.Buffer
		for {
			select {
			case chunk, ok := <-stream:
				if !ok {
					return
				}
				buf.Write(chunk.Body)
				for {
					line, err := buf.ReadString('\n')
					if err != nil {
						// not enough for a full line yet
						if err == io.EOF {
							break
						}
						// unexpected error
						errChan <- fmt.Errorf("failed to read stream: %w", err)
						return
					}

					line = strings.TrimRight(line, "\r\n")
					if line == "" || strings.HasPrefix(line, ":") {
						continue
					}
					if !strings.HasPrefix(line, "data: ") {
						continue
					}
					data := strings.TrimPrefix(line, "data: ")
					if data == "[DONE]" {
						return
					}
					var streamResp QWenStreamResponse
					if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
						continue
					}
					if len(streamResp.Choices) > 0 {
						choice := streamResp.Choices[0]
						if choice.Delta.Content != "" {
							msgChan <- &schema.Message{
								Role:    schema.RoleAssistant,
								Content: choice.Delta.Content,
							}
						}
					}
				}
			case err, ok := <-errs:
				if !ok {
					return
				}
				if err != nil {
					errChan <- fmt.Errorf("failed to read stream: %w", err)
					return
				}
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return msgChan, errChan
}
