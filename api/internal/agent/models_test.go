package agent

// Model caching tests - commented out, not yet ready for integration.
/*
import (
	"testing"
	"time"
)

// AvailableModels tests - commented out, function is in commented-out openrouter.go

func TestAvailableModels_NonEmpty(t *testing.T) {
	models := AvailableModels()
	if len(models) == 0 {
		t.Fatal("expected non-empty list of models")
	}
}

func TestAvailableModels_HasExpectedFields(t *testing.T) {
	models := AvailableModels()
	for i, m := range models {
		if _, ok := m["id"]; !ok {
			t.Errorf("model %d: missing 'id' field", i)
		}
		if _, ok := m["name"]; !ok {
			t.Errorf("model %d: missing 'name' field", i)
		}
		if _, ok := m["input_cost_per_1k"]; !ok {
			t.Errorf("model %d: missing 'input_cost_per_1k' field", i)
		}
		if _, ok := m["output_cost_per_1k"]; !ok {
			t.Errorf("model %d: missing 'output_cost_per_1k' field", i)
		}
	}
}

func TestAvailableModels_ContainsClaude(t *testing.T) {
	models := AvailableModels()
	found := false
	for _, m := range models {
		if id, ok := m["id"].(string); ok && id == "anthropic/claude-sonnet-4" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected AvailableModels to contain anthropic/claude-sonnet-4")
	}
}

func TestFallbackModels_NonEmpty(t *testing.T) {
	models := fallbackModels()
	if len(models) == 0 {
		t.Fatal("expected non-empty fallback models")
	}
}

func TestFallbackModels_AllHaveToolCall(t *testing.T) {
	models := fallbackModels()
	for _, m := range models {
		if !m.ToolCall {
			t.Errorf("model %s: expected ToolCall=true", m.ID)
		}
	}
}

func TestFallbackModels_FieldsPopulated(t *testing.T) {
	models := fallbackModels()
	for _, m := range models {
		if m.ID == "" {
			t.Error("expected non-empty ID")
		}
		if m.Name == "" {
			t.Errorf("model %s: expected non-empty Name", m.ID)
		}
		if m.InputCostPer1K <= 0 {
			t.Errorf("model %s: expected positive InputCostPer1K, got %f", m.ID, m.InputCostPer1K)
		}
		if m.OutputCostPer1K <= 0 {
			t.Errorf("model %s: expected positive OutputCostPer1K, got %f", m.ID, m.OutputCostPer1K)
		}
		if m.ContextLimit <= 0 {
			t.Errorf("model %s: expected positive ContextLimit, got %d", m.ID, m.ContextLimit)
		}
		if m.OutputLimit <= 0 {
			t.Errorf("model %s: expected positive OutputLimit, got %d", m.ID, m.OutputLimit)
		}
	}
}

func TestNewModelCache(t *testing.T) {
	mc := NewModelCache(5 * time.Minute)
	if mc == nil {
		t.Fatal("expected non-nil ModelCache")
	}
	if mc.ttl != 5*time.Minute {
		t.Errorf("expected ttl 5m, got %v", mc.ttl)
	}
}

func TestModelCache_GetModels_ReturnsFallbackWhenEmpty(t *testing.T) {
	// Use a very short TTL so cache is always stale
	mc := NewModelCache(0)

	// GetModels will try to fetch from OpenRouter (which will fail in tests)
	// and should fall back to fallbackModels
	models, err := mc.GetModels(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected non-empty models from fallback")
	}
}

func TestModelCache_GetModels_ReturnsCopy(t *testing.T) {
	mc := NewModelCache(0)

	models1, _ := mc.GetModels(t.Context())
	models2, _ := mc.GetModels(t.Context())

	if len(models1) == 0 || len(models2) == 0 {
		t.Skip("no models returned, cannot test copy behavior")
	}

	// Modifying one should not affect the other
	models1[0].Name = "mutated"
	if models2[0].Name == "mutated" {
		t.Error("expected GetModels to return independent copies")
	}
}
*/
