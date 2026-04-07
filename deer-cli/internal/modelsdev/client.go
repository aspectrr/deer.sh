package modelsdev

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Model represents a model from the OpenRouter API.
type Model struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	InputCostPer1M  float64 `json:"input_cost_per_1m"`
	OutputCostPer1M float64 `json:"output_cost_per_1m"`
	ContextLimit    int     `json:"context_limit"`
	OutputLimit     int     `json:"output_limit"`
	ToolCall        bool    `json:"tool_call"`
	Reasoning       bool    `json:"reasoning"`
}

type openRouterResponse struct {
	Data []openRouterEntry `json:"data"`
}

type openRouterEntry struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	ContextLength       int             `json:"context_length"`
	Pricing             openRouterPrice `json:"pricing"`
	TopProvider         *openRouterTop  `json:"top_provider"`
	SupportedParameters []string        `json:"supported_parameters"`
}

type openRouterPrice struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

type openRouterTop struct {
	ContextLength       int `json:"context_length"`
	MaxCompletionTokens int `json:"max_completion_tokens"`
}

// FetchTopModels fetches models from OpenRouter, groups by provider prefix,
// and returns the top model per provider (highest input cost with tool_call support).
func FetchTopModels(ctx context.Context) ([]Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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
		return nil, fmt.Errorf("openrouter API returned %d", resp.StatusCode)
	}

	var data openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	type candidate struct {
		entry      openRouterEntry
		inputPer1M float64
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

		inputPer1M := inputPerToken * 1_000_000

		existing, ok := topByProvider[provider]
		if !ok || inputPer1M > existing.inputPer1M {
			topByProvider[provider] = candidate{entry: entry, inputPer1M: inputPer1M}
		}
	}

	models := make([]Model, 0, len(topByProvider))
	for _, c := range topByProvider {
		outputPerToken, _ := strconv.ParseFloat(c.entry.Pricing.Completion, 64)

		outputLimit := 0
		if c.entry.TopProvider != nil {
			outputLimit = c.entry.TopProvider.MaxCompletionTokens
		}

		hasReasoning := slices.Contains(c.entry.SupportedParameters, "reasoning") ||
			slices.Contains(c.entry.SupportedParameters, "include_reasoning")

		models = append(models, Model{
			ID:              c.entry.ID,
			Name:            c.entry.Name,
			InputCostPer1M:  c.inputPer1M,
			OutputCostPer1M: outputPerToken * 1_000_000,
			ContextLimit:    c.entry.ContextLength,
			OutputLimit:     outputLimit,
			ToolCall:        true,
			Reasoning:       hasReasoning,
		})
	}

	// Sort by input cost descending
	slices.SortFunc(models, func(a, b Model) int {
		if a.InputCostPer1M > b.InputCostPer1M {
			return -1
		}
		if a.InputCostPer1M < b.InputCostPer1M {
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})

	if len(models) == 0 {
		return nil, fmt.Errorf("no models with tool_call support found")
	}

	return models, nil
}
