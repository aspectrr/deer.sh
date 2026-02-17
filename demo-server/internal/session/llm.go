package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// LLMMessage is a message in the conversation history (OpenAI format).
type LLMMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall represents a tool call from the LLM.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall holds the function name and arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAITool is a tool in OpenAI function-calling format.
type OpenAITool struct {
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

// OpenAIFunction defines a function for OpenAI tool use.
type OpenAIFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// chatRequest is the OpenAI-compatible chat completion request.
type chatRequest struct {
	Model    string       `json:"model"`
	Messages []LLMMessage `json:"messages"`
	Tools    []OpenAITool `json:"tools,omitempty"`
}

// chatResponse is the OpenAI-compatible chat completion response.
type chatResponse struct {
	ID      string   `json:"id"`
	Choices []choice `json:"choices"`
}

type choice struct {
	Message      LLMMessage `json:"message"`
	FinishReason string     `json:"finish_reason"`
}

// LLMClient calls an OpenRouter-compatible API.
type LLMClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewLLMClient creates a new LLM client.
func NewLLMClient(apiKey, model string) *LLMClient {
	return &LLMClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Chat sends a chat completion request and returns the response.
func (c *LLMClient) Chat(ctx context.Context, messages []LLMMessage, tools []OpenAITool) (*chatResponse, error) {
	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
	}
	if len(tools) > 0 {
		reqBody.Tools = tools
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &chatResp, nil
}

// convertMCPToolsToOpenAI converts MCP tool definitions to OpenAI function-calling format.
func convertMCPToolsToOpenAI(tools []mcp.Tool) []OpenAITool {
	result := make([]OpenAITool, 0, len(tools))
	for _, t := range tools {
		oaiTool := OpenAITool{
			Type: "function",
			Function: OpenAIFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		}
		result = append(result, oaiTool)
	}
	return result
}
