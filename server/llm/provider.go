package llm

import (
	"fmt"
	"strings"
)

// ScenarioGenerator produces a scenario JSON document from a teacher prompt.
// It intentionally excludes tool calling, which remains an OpenRouter-only
// capability for the separate agent chat endpoint.
type ScenarioGenerator interface {
	Generate(prompt string, schemaJSON []byte, registryJSON []byte) (string, error)
	IsConfigured() bool
}

// ScenarioRepairer optionally retries a rejected generated document using the
// validator's exact errors. The repaired result must still pass validation.
type ScenarioRepairer interface {
	Repair(invalidJSON string, details []string, schemaJSON []byte, registryJSON []byte) (string, error)
}

// NewScenarioGenerator selects the provider used by POST /api/scenario/generate.
// OpenRouter remains the default so existing deployments need no changes.
func NewScenarioGenerator() (ScenarioGenerator, error) {
	switch strings.ToLower(strings.TrimSpace(envOr("LLM_PROVIDER"))) {
	case "", "openrouter":
		return NewClient(), nil
	case "ollama":
		return NewOllamaClient(), nil
	default:
		return nil, fmt.Errorf("unsupported LLM_PROVIDER %q; use openrouter or ollama", envOr("LLM_PROVIDER"))
	}
}

func (c *OpenRouterClient) IsConfigured() bool {
	return strings.TrimSpace(c.APIKey) != ""
}
