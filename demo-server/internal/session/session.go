package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	maxCommandsPerSession  = 50
	maxSandboxesPerSession = 1
)

// Config holds session configuration.
type Config struct {
	FluidBin       string
	LLMAPIKey      string
	LLMModel       string
	SessionTimeout time.Duration
	Logger         *slog.Logger
}

// Event is a message sent from the session to the browser.
type Event struct {
	Type     string      `json:"type"`
	Content  string      `json:"content,omitempty"`
	ToolName string      `json:"tool_name,omitempty"`
	Args     interface{} `json:"args,omitempty"`
	Success  *bool       `json:"success,omitempty"`
	Result   interface{} `json:"result,omitempty"`
	Active   *bool       `json:"active,omitempty"`
	Message  string      `json:"message,omitempty"`
	// session_info fields
	SessionID    string `json:"session_id,omitempty"`
	ExpiresInSec int    `json:"expires_in_sec,omitempty"`
}

// DemoSession manages one browser connection's interaction with the fluid MCP server.
type DemoSession struct {
	ID           string
	config       Config
	mcpClient    *mcpclient.Client
	llm          *LLMClient
	tools        []mcp.Tool
	openAITools  []OpenAITool
	messages     []LLMMessage
	sandboxIDs   []string
	commandCount int
	startTime    time.Time
	lastActivity time.Time
	eventCh      chan Event
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.Mutex
	closed       bool
	logger       *slog.Logger
}

// New creates a new DemoSession. It spawns the fluid MCP subprocess and initializes the connection.
func New(cfg Config) (*DemoSession, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &DemoSession{
		ID:           uuid.New().String()[:8],
		config:       cfg,
		llm:          NewLLMClient(cfg.LLMAPIKey, cfg.LLMModel),
		startTime:    time.Now(),
		lastActivity: time.Now(),
		eventCh:      make(chan Event, 64),
		ctx:          ctx,
		cancel:       cancel,
		logger:       cfg.Logger.With("session_id", ""),
	}
	s.logger = cfg.Logger.With("session_id", s.ID)

	// Spawn fluid mcp subprocess
	c, err := mcpclient.NewStdioMCPClient(cfg.FluidBin, nil, "mcp")
	if err != nil {
		cancel()
		return nil, fmt.Errorf("spawn fluid mcp: %w", err)
	}
	s.mcpClient = c

	// Initialize MCP connection
	initReq := mcp.InitializeRequest{}
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "demo-server",
		Version: "1.0.0",
	}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	_, err = s.mcpClient.Initialize(ctx, initReq)
	if err != nil {
		c.Close()
		cancel()
		return nil, fmt.Errorf("mcp initialize: %w", err)
	}

	// Discover available tools
	toolsResult, err := s.mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		c.Close()
		cancel()
		return nil, fmt.Errorf("mcp list tools: %w", err)
	}
	s.tools = toolsResult.Tools
	s.openAITools = convertMCPToolsToOpenAI(s.tools)

	// Set system prompt
	s.messages = []LLMMessage{
		{
			Role: "system",
			Content: `You are a fluid.sh demo agent helping users try out VM sandbox management. You have access to fluid tools for creating sandboxes, running commands, editing files, and creating Ansible playbooks.

Be concise and helpful. Show what fluid can do. When users ask you to do something, use the available tools to accomplish it.

Important constraints:
- You can create at most 1 sandbox per session
- Keep commands focused and demonstrative
- After completing a task, briefly summarize what you did`,
		},
	}

	s.logger.Info("session created", "tools", len(s.tools))
	return s, nil
}

// Events returns the channel for reading session events.
func (s *DemoSession) Events() <-chan Event {
	return s.eventCh
}

// HandleMessage processes a user message through the LLM agent loop.
func (s *DemoSession) HandleMessage(userMsg string) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.lastActivity = time.Now()
	s.mu.Unlock()

	s.messages = append(s.messages, LLMMessage{Role: "user", Content: userMsg})

	// Send thinking indicator
	active := true
	s.sendEvent(Event{Type: "thinking", Active: &active})

	// Agent loop: send to LLM, execute tool calls, repeat until text-only response
	for {
		if s.ctx.Err() != nil {
			return
		}

		resp, err := s.llm.Chat(s.ctx, s.messages, s.openAITools)
		if err != nil {
			s.sendEvent(Event{Type: "error", Message: fmt.Sprintf("LLM error: %v", err)})
			return
		}

		if len(resp.Choices) == 0 {
			s.sendEvent(Event{Type: "error", Message: "empty LLM response"})
			return
		}

		choice := resp.Choices[0]
		msg := choice.Message

		// No tool calls - send text response and break
		if len(msg.ToolCalls) == 0 {
			inactive := false
			s.sendEvent(Event{Type: "thinking", Active: &inactive})
			if msg.Content != "" {
				s.sendEvent(Event{Type: "assistant_message", Content: msg.Content})
			}
			s.messages = append(s.messages, LLMMessage{Role: "assistant", Content: msg.Content})
			break
		}

		// Has tool calls - execute them
		assistantMsg := LLMMessage{
			Role:      "assistant",
			Content:   msg.Content,
			ToolCalls: msg.ToolCalls,
		}
		s.messages = append(s.messages, assistantMsg)

		if msg.Content != "" {
			s.sendEvent(Event{Type: "assistant_message", Content: msg.Content})
		}

		for _, tc := range msg.ToolCalls {
			if err := s.executeToolCall(tc); err != nil {
				s.logger.Error("tool call failed", "tool", tc.Function.Name, "error", err)
			}
		}
	}
}

// extractText gets text content from MCP Content interface slice.
func extractText(contents []mcp.Content) string {
	var result string
	for _, c := range contents {
		if tc, ok := mcp.AsTextContent(c); ok {
			result += tc.Text
		}
	}
	return result
}

// executeToolCall runs a single tool call via MCP and sends events.
func (s *DemoSession) executeToolCall(tc ToolCall) error {
	// Parse arguments
	var args map[string]interface{}
	if tc.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			args = map[string]interface{}{}
		}
	}

	// Check limits
	s.mu.Lock()
	s.commandCount++
	if s.commandCount > maxCommandsPerSession {
		s.mu.Unlock()
		errMsg := "command limit reached for this session"
		s.sendEvent(Event{Type: "error", Message: errMsg})
		s.messages = append(s.messages, LLMMessage{
			Role:       "tool",
			Content:    errMsg,
			ToolCallID: tc.ID,
		})
		return fmt.Errorf(errMsg)
	}
	if tc.Function.Name == "create_sandbox" && len(s.sandboxIDs) >= maxSandboxesPerSession {
		s.mu.Unlock()
		errMsg := "sandbox limit reached (max 1 per session)"
		s.sendEvent(Event{Type: "error", Message: errMsg})
		s.messages = append(s.messages, LLMMessage{
			Role:       "tool",
			Content:    errMsg,
			ToolCallID: tc.ID,
		})
		return fmt.Errorf(errMsg)
	}
	s.mu.Unlock()

	// Send tool_start event
	s.sendEvent(Event{Type: "tool_start", ToolName: tc.Function.Name, Args: args})

	// Execute via MCP
	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = tc.Function.Name
	callReq.Params.Arguments = args

	result, err := s.mcpClient.CallTool(s.ctx, callReq)

	var resultText string
	success := true

	if err != nil {
		success = false
		resultText = fmt.Sprintf("error: %v", err)
	} else if result.IsError {
		success = false
		resultText = extractText(result.Content)
		if resultText == "" {
			resultText = "tool returned error"
		}
	} else {
		resultText = extractText(result.Content)
	}

	// Track created sandboxes
	if tc.Function.Name == "create_sandbox" && success {
		var createResult map[string]interface{}
		if err := json.Unmarshal([]byte(resultText), &createResult); err == nil {
			if sid, ok := createResult["sandbox_id"].(string); ok {
				s.mu.Lock()
				s.sandboxIDs = append(s.sandboxIDs, sid)
				s.mu.Unlock()
			}
		}
	}

	// Parse result for display
	var resultDisplay interface{}
	var parsed interface{}
	if err := json.Unmarshal([]byte(resultText), &parsed); err == nil {
		resultDisplay = parsed
	} else {
		resultDisplay = resultText
	}

	s.sendEvent(Event{
		Type:     "tool_complete",
		ToolName: tc.Function.Name,
		Success:  &success,
		Result:   resultDisplay,
	})

	// Add tool result to conversation
	s.messages = append(s.messages, LLMMessage{
		Role:       "tool",
		Content:    resultText,
		ToolCallID: tc.ID,
	})

	return nil
}

// Close cleans up the session: destroys sandboxes, kills MCP subprocess.
func (s *DemoSession) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	sandboxes := make([]string, len(s.sandboxIDs))
	copy(sandboxes, s.sandboxIDs)
	s.mu.Unlock()

	s.logger.Info("closing session", "sandboxes", len(sandboxes))

	// Destroy any created sandboxes
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, sid := range sandboxes {
		req := mcp.CallToolRequest{}
		req.Params.Name = "destroy_sandbox"
		req.Params.Arguments = map[string]interface{}{"sandbox_id": sid}
		if _, err := s.mcpClient.CallTool(ctx, req); err != nil {
			s.logger.Error("failed to destroy sandbox", "sandbox_id", sid, "error", err)
		} else {
			s.logger.Info("destroyed sandbox", "sandbox_id", sid)
		}
	}

	s.cancel()
	if s.mcpClient != nil {
		s.mcpClient.Close()
	}
	close(s.eventCh)
}

// IsExpired returns true if the session has exceeded the inactivity timeout.
func (s *DemoSession) IsExpired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Since(s.lastActivity) > s.config.SessionTimeout
}

// ExpiresInSec returns seconds until session expiry.
func (s *DemoSession) ExpiresInSec() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	remaining := s.config.SessionTimeout - time.Since(s.lastActivity)
	if remaining < 0 {
		return 0
	}
	return int(remaining.Seconds())
}

func (s *DemoSession) sendEvent(e Event) {
	select {
	case s.eventCh <- e:
	case <-s.ctx.Done():
	}
}
