package agent

// Model caching - commented out, not yet ready for integration.
/*
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ModelInfo represents a model with pricing and capability info.
type ModelInfo struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	InputCostPer1K  float64 `json:"input_cost_per_1k"`
	OutputCostPer1K float64 `json:"output_cost_per_1k"`
	ContextLimit    int     `json:"context_limit"`
	OutputLimit     int     `json:"output_limit"`
	ToolCall        bool    `json:"tool_call"`
	Reasoning       bool    `json:"reasoning"`
}

// ModelCache caches the list of available models with a TTL.
type ModelCache struct {
	mu        sync.RWMutex
	models    []ModelInfo
	fetchedAt time.Time
	ttl       time.Duration
}

// NewModelCache creates a new model cache with the given TTL.
func NewModelCache(ttl time.Duration) *ModelCache {
	return &ModelCache{ttl: ttl}
}

// GetModels returns cached models or re-fetches if the cache is stale.
func (mc *ModelCache) GetModels(ctx context.Context) ([]ModelInfo, error) {
	mc.mu.RLock()
	if len(mc.models) > 0 && time.Since(mc.fetchedAt) < mc.ttl {
		models := make([]ModelInfo, len(mc.models))
		copy(models, mc.models)
		mc.mu.RUnlock()
		return models, nil
	}
	mc.mu.RUnlock()

	models, err := fetchFromOpenRouter(ctx)
	if err != nil {
		// Return fallback if cache is empty
		mc.mu.RLock()
		defer mc.mu.RUnlock()
		if len(mc.models) > 0 {
			result := make([]ModelInfo, len(mc.models))
			copy(result, mc.models)
			return result, nil
		}
		return fallbackModels(), nil
	}

	mc.mu.Lock()
	mc.models = models
	mc.fetchedAt = time.Now()
	mc.mu.Unlock()

	result := make([]ModelInfo, len(models))
	copy(result, models)
	return result, nil
}

// openRouterModelsResponse is the response from GET /api/v1/models.
type openRouterModelsResponse struct {
	Data []openRouterModelEntry `json:"data"`
}

type openRouterModelEntry struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	ContextLength       int               `json:"context_length"`
	Pricing             openRouterPricing `json:"pricing"`
	TopProvider         *openRouterTop    `json:"top_provider"`
	SupportedParameters []string          `json:"supported_parameters"`
}

type openRouterPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

type openRouterTop struct {
	ContextLength       int `json:"context_length"`
	MaxCompletionTokens int `json:"max_completion_tokens"`
}

// fetchFromOpenRouter fetches models from OpenRouter's public API,
// groups by provider prefix, and returns the top model per provider.
func fetchFromOpenRouter(ctx context.Context) ([]ModelInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter models API returned %d", resp.StatusCode)
	}

	var data openRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Group by provider prefix, pick top model per provider
	// "top" = highest input cost among models with tool_call support
	type candidate struct {
		entry      openRouterModelEntry
		inputPer1K float64
	}

	topByProvider := make(map[string]candidate)

	for _, entry := range data.Data {
		parts := strings.SplitN(entry.ID, "/", 2)
		if len(parts) != 2 {
			continue
		}
		provider := parts[0]

		hasToolCall := slices.Contains(entry.SupportedParameters, "tool_choice")
		if !hasToolCall {
			continue
		}

		inputPerToken, err := strconv.ParseFloat(entry.Pricing.Prompt, 64)
		if err != nil || inputPerToken <= 0 {
			continue
		}

		inputPer1K := inputPerToken * 1000.0

		existing, ok := topByProvider[provider]
		if !ok || inputPer1K > existing.inputPer1K {
			topByProvider[provider] = candidate{entry: entry, inputPer1K: inputPer1K}
		}
	}

	models := make([]ModelInfo, 0, len(topByProvider))
	for _, c := range topByProvider {
		outputPerToken, _ := strconv.ParseFloat(c.entry.Pricing.Completion, 64)

		outputLimit := 0
		if c.entry.TopProvider != nil {
			outputLimit = c.entry.TopProvider.MaxCompletionTokens
		}

		hasReasoning := slices.Contains(c.entry.SupportedParameters, "reasoning") ||
			slices.Contains(c.entry.SupportedParameters, "include_reasoning")

		models = append(models, ModelInfo{
			ID:              c.entry.ID,
			Name:            c.entry.Name,
			InputCostPer1K:  c.inputPer1K,
			OutputCostPer1K: outputPerToken * 1000.0,
			ContextLimit:    c.entry.ContextLength,
			OutputLimit:     outputLimit,
			ToolCall:        true,
			Reasoning:       hasReasoning,
		})
	}

	// Sort by input cost descending for consistent ordering
	slices.SortFunc(models, func(a, b ModelInfo) int {
		if a.InputCostPer1K > b.InputCostPer1K {
			return -1
		}
		if a.InputCostPer1K < b.InputCostPer1K {
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})

	if len(models) == 0 {
		return nil, fmt.Errorf("no models with tool_call support found")
	}

	return models, nil
}

// fallbackModels returns hardcoded models when the API is unreachable.
func fallbackModels() []ModelInfo {
	return []ModelInfo{
		{ID: "anthropic/claude-sonnet-4", Name: "Claude Sonnet 4", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextLimit: 200000, OutputLimit: 64000, ToolCall: true, Reasoning: false},
		{ID: "anthropic/claude-haiku-4", Name: "Claude Haiku 4", InputCostPer1K: 0.0008, OutputCostPer1K: 0.004, ContextLimit: 200000, OutputLimit: 64000, ToolCall: true, Reasoning: false},
		{ID: "openai/gpt-4o", Name: "GPT-4o", InputCostPer1K: 0.0025, OutputCostPer1K: 0.01, ContextLimit: 128000, OutputLimit: 16384, ToolCall: true, Reasoning: false},
		{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini", InputCostPer1K: 0.00015, OutputCostPer1K: 0.0006, ContextLimit: 128000, OutputLimit: 16384, ToolCall: true, Reasoning: false},
		{ID: "google/gemini-2.5-pro", Name: "Gemini 2.5 Pro", InputCostPer1K: 0.00125, OutputCostPer1K: 0.01, ContextLimit: 1000000, OutputLimit: 65536, ToolCall: true, Reasoning: true},
	}
}
*/
