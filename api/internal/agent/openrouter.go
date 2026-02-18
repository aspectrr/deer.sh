package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/config"
	"github.com/aspectrr/fluid.sh/api/internal/orchestrator"
	"github.com/aspectrr/fluid.sh/api/internal/store"
	"github.com/google/uuid"
)

const maxToolCalls = 20

// Client communicates with OpenRouter and executes tool calls.
type Client struct {
	httpClient   *http.Client
	apiKey       string
	baseURL      string
	defaultModel string
	maxTokens    int
	store        store.DataStore
	orchestrator *orchestrator.Orchestrator
	logger       *slog.Logger
}

// NewClient creates a new agent client.
func NewClient(cfg config.AgentConfig, st store.DataStore, orch *orchestrator.Orchestrator, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		httpClient:   &http.Client{Timeout: 5 * time.Minute},
		apiKey:       cfg.OpenRouterAPIKey,
		baseURL:      cfg.OpenRouterBaseURL,
		defaultModel: cfg.DefaultModel,
		maxTokens:    cfg.MaxTokensPerRequest,
		store:        st,
		orchestrator: orch,
		logger:       logger.With("component", "agent"),
	}
}

// ChatRequest is the input for a chat stream.
type ChatRequest struct {
	ConversationID string `json:"conversation_id,omitempty"`
	Message        string `json:"message"`
	Model          string `json:"model,omitempty"`
}

// openRouterRequest is the request body sent to OpenRouter.
type openRouterRequest struct {
	Model     string          `json:"model"`
	Messages  []openRouterMsg `json:"messages"`
	Tools     []Tool          `json:"tools,omitempty"`
	Stream    bool            `json:"stream"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openRouterMsg struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// StreamChat handles the full agentic loop: sends to OpenRouter, streams back SSE,
// handles tool calls, and persists messages.
func (c *Client) StreamChat(ctx context.Context, w http.ResponseWriter, orgID, userID string, req ChatRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	model := req.Model
	if model == "" {
		model = c.defaultModel
	}

	// Load or create conversation
	var conv *store.AgentConversation
	if req.ConversationID != "" {
		var err error
		conv, err = c.store.GetAgentConversation(ctx, req.ConversationID)
		if err != nil {
			c.writeSSE(w, flusher, "error", map[string]string{"error": "conversation not found"})
			return
		}
	} else {
		conv = &store.AgentConversation{
			ID:     uuid.New().String(),
			OrgID:  orgID,
			UserID: userID,
			Title:  truncate(req.Message, 100),
			Model:  model,
		}
		if err := c.store.CreateAgentConversation(ctx, conv); err != nil {
			c.writeSSE(w, flusher, "error", map[string]string{"error": "failed to create conversation"})
			return
		}
	}

	// Build messages from history
	history, _ := c.store.ListAgentMessages(ctx, conv.ID)
	messages := make([]openRouterMsg, 0, len(history)+1)
	messages = append(messages, openRouterMsg{
		Role:    "system",
		Content: "You are a helpful infrastructure AI agent for fluid.sh. You can manage sandboxes, source VMs, hosts, and playbooks. Use tools to perform actions. Be concise and precise.",
	})
	for _, m := range history {
		msg := openRouterMsg{
			Role:    string(m.Role),
			Content: m.Content,
		}
		if m.ToolCalls != "" {
			msg.ToolCalls = json.RawMessage(m.ToolCalls)
		}
		if m.ToolCallID != "" {
			msg.ToolCallID = m.ToolCallID
		}
		messages = append(messages, msg)
	}

	// Add the new user message
	userMsg := &store.AgentMessage{
		ID:             uuid.New().String(),
		ConversationID: conv.ID,
		Role:           store.MessageRoleUser,
		Content:        req.Message,
		Model:          model,
	}
	if err := c.store.CreateAgentMessage(ctx, userMsg); err != nil {
		c.logger.Warn("failed to persist user message", "conversation_id", conv.ID, "error", err)
	}
	messages = append(messages, openRouterMsg{Role: "user", Content: req.Message})

	// Send message_start
	msgID := uuid.New().String()
	c.writeSSE(w, flusher, "message_start", map[string]string{
		"conversation_id": conv.ID,
		"message_id":      msgID,
	})

	totalInputTokens := 0
	totalOutputTokens := 0
	toolCallCount := 0

	// Agentic loop
	for {
		// Break if client disconnected
		if ctx.Err() != nil {
			c.logger.Info("client disconnected", "conversation_id", conv.ID)
			break
		}

		if toolCallCount >= maxToolCalls {
			c.logger.Warn("max tool calls reached", "conversation_id", conv.ID)
			break
		}

		content, toolCalls, inputTokens, outputTokens, err := c.callOpenRouter(ctx, w, flusher, model, messages)
		totalInputTokens += inputTokens
		totalOutputTokens += outputTokens

		if err != nil {
			c.writeSSE(w, flusher, "error", map[string]string{"error": err.Error()})
			break
		}

		if len(toolCalls) == 0 {
			// No tool calls - persist assistant message and finish
			assistantMsg := &store.AgentMessage{
				ID:             msgID,
				ConversationID: conv.ID,
				Role:           store.MessageRoleAssistant,
				Content:        content,
				TokensInput:    totalInputTokens,
				TokensOutput:   totalOutputTokens,
				Model:          model,
			}
			if err := c.store.CreateAgentMessage(ctx, assistantMsg); err != nil {
				c.logger.Warn("failed to persist assistant message", "conversation_id", conv.ID, "error", err)
			}
			break
		}

		// Process tool calls
		toolCallsJSON, _ := json.Marshal(toolCalls)
		assistantMsg := &store.AgentMessage{
			ID:             uuid.New().String(),
			ConversationID: conv.ID,
			Role:           store.MessageRoleAssistant,
			Content:        content,
			ToolCalls:      string(toolCallsJSON),
			Model:          model,
		}
		if err := c.store.CreateAgentMessage(ctx, assistantMsg); err != nil {
			c.logger.Warn("failed to persist assistant tool-call message", "conversation_id", conv.ID, "error", err)
		}

		messages = append(messages, openRouterMsg{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCallsJSON,
		})

		for _, tc := range toolCalls {
			toolCallCount++
			c.writeSSE(w, flusher, "tool_call_start", map[string]string{
				"tool_call_id": tc.ID,
				"name":         tc.Function.Name,
			})

			result, _ := c.ExecuteTool(ctx, orgID, tc.Function.Name, json.RawMessage(tc.Function.Arguments))

			c.writeSSE(w, flusher, "tool_result", map[string]any{
				"tool_call_id": tc.ID,
				"result":       result,
			})

			// Persist tool result message
			toolMsg := &store.AgentMessage{
				ID:             uuid.New().String(),
				ConversationID: conv.ID,
				Role:           store.MessageRoleTool,
				Content:        result,
				ToolCallID:     tc.ID,
				Model:          model,
			}
			if err := c.store.CreateAgentMessage(ctx, toolMsg); err != nil {
				c.logger.Warn("failed to persist tool result message", "conversation_id", conv.ID, "error", err)
			}

			messages = append(messages, openRouterMsg{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	// Record token usage
	if totalInputTokens+totalOutputTokens > 0 {
		_ = c.store.CreateUsageRecord(ctx, &store.UsageRecord{
			ID:           uuid.New().String(),
			OrgID:        orgID,
			ResourceType: "llm_token",
			Quantity:     float64(totalInputTokens + totalOutputTokens),
			MetadataJSON: fmt.Sprintf(`{"model":"%s","input":%d,"output":%d}`, model, totalInputTokens, totalOutputTokens),
		})
	}

	c.writeSSE(w, flusher, "message_end", map[string]any{
		"tokens_input":  totalInputTokens,
		"tokens_output": totalOutputTokens,
	})
	c.writeSSE(w, flusher, "done", "[DONE]")
}

// toolCall represents a tool call from the LLM response.
type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// callOpenRouter sends a request to OpenRouter and streams the response.
// Returns the accumulated content, any tool calls, and token counts.
func (c *Client) callOpenRouter(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, model string, messages []openRouterMsg) (string, []toolCall, int, int, error) {
	reqBody := openRouterRequest{
		Model:     model,
		Messages:  messages,
		Tools:     AllTools(),
		Stream:    true,
		MaxTokens: c.maxTokens,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, 0, 0, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", nil, 0, 0, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", nil, 0, 0, fmt.Errorf("openrouter request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", nil, 0, 0, fmt.Errorf("openrouter error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse SSE stream from OpenRouter
	var contentBuilder strings.Builder
	var toolCalls []toolCall
	toolCallArgBuilders := map[int]*strings.Builder{}
	inputTokens := 0
	outputTokens := 0

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			inputTokens = chunk.Usage.PromptTokens
			outputTokens = chunk.Usage.CompletionTokens
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// Content delta
		if delta.Content != "" {
			contentBuilder.WriteString(delta.Content)
			c.writeSSE(w, flusher, "content_delta", map[string]string{
				"delta": delta.Content,
			})
		}

		// Tool call deltas
		for _, tc := range delta.ToolCalls {
			if tc.ID != "" {
				// New tool call
				for len(toolCalls) <= tc.Index {
					toolCalls = append(toolCalls, toolCall{})
				}
				toolCalls[tc.Index].ID = tc.ID
				toolCalls[tc.Index].Type = tc.Type
				toolCalls[tc.Index].Function.Name = tc.Function.Name
				toolCallArgBuilders[tc.Index] = &strings.Builder{}
			}
			if tc.Function.Arguments != "" {
				if builder, ok := toolCallArgBuilders[tc.Index]; ok {
					builder.WriteString(tc.Function.Arguments)
				}
			}
		}
	}

	// Finalize tool call arguments
	for i, builder := range toolCallArgBuilders {
		if i < len(toolCalls) {
			toolCalls[i].Function.Arguments = builder.String()
		}
	}

	return contentBuilder.String(), toolCalls, inputTokens, outputTokens, nil
}

func (c *Client) writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, data any) {
	dataBytes, _ := json.Marshal(data)
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(dataBytes))
	flusher.Flush()
}

func truncate(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen])
}

// AvailableModels returns the list of models with pricing info.
func AvailableModels() []map[string]any {
	return []map[string]any{
		{"id": "anthropic/claude-sonnet-4", "name": "Claude Sonnet 4", "input_cost_per_1k": 0.003, "output_cost_per_1k": 0.015},
		{"id": "anthropic/claude-haiku-4", "name": "Claude Haiku 4", "input_cost_per_1k": 0.0008, "output_cost_per_1k": 0.004},
		{"id": "openai/gpt-4o", "name": "GPT-4o", "input_cost_per_1k": 0.0025, "output_cost_per_1k": 0.01},
		{"id": "openai/gpt-4o-mini", "name": "GPT-4o Mini", "input_cost_per_1k": 0.00015, "output_cost_per_1k": 0.0006},
		{"id": "google/gemini-2.5-pro", "name": "Gemini 2.5 Pro", "input_cost_per_1k": 0.00125, "output_cost_per_1k": 0.01},
	}
}
