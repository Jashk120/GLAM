package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"glam/server/llm"
	scenario_pkg "glam/server/scenario"
)

func GenerateScenarioWorld(ctx context.Context, deps Deps, topic, ageGroup, templateHint string, llmClient *llm.OpenRouterClient) (map[string]interface{}, []string, error) {
	structure, err := GenerateStructure(ctx, llmClient, deps, topic, ageGroup, templateHint)
	if err != nil {
		return nil, nil, fmt.Errorf("structure: %w", err)
	}
	flavor, err := GenerateFlavor(ctx, llmClient, deps, structure, topic)
	if err != nil {
		return nil, nil, fmt.Errorf("flavor: %w", err)
	}
	interactions, err := GenerateInteractions(ctx, llmClient, deps, structure, flavor, topic, ageGroup)
	if err != nil {
		return nil, nil, fmt.Errorf("interactions: %w", err)
	}
	obj, err := Assemble(structure, flavor, interactions, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("assemble: %w", err)
	}
	warnings := []string{}
	raw, _ := json.Marshal(obj)
	ok, details, err := scenario_pkg.ValidateScenario(raw, deps.SchemaPath, deps.RegistryPath)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("validation internal error: %v", err))
		return obj, warnings, nil
	}
	if !ok {
		if fixed, didFix, _ := scenario_pkg.NormalizePlotRefs(raw); didFix {
			if ok2, details2, _ := scenario_pkg.ValidateScenario(fixed, deps.SchemaPath, deps.RegistryPath); ok2 {
				raw = fixed
				var fixedObj map[string]interface{}
				_ = json.Unmarshal(raw, &fixedObj)
				return fixedObj, warnings, nil
			} else {
				_ = details2
			}
		}
		if sanitized, didSan, _ := scenario_pkg.SanitizeExtraFields(raw); didSan {
			if ok2, _, _ := scenario_pkg.ValidateScenario(sanitized, deps.SchemaPath, deps.RegistryPath); ok2 {
				var cleanObj map[string]interface{}
				_ = json.Unmarshal(sanitized, &cleanObj)
				return cleanObj, warnings, nil
			}
		}
		warnings = append(warnings, details...)
	}
	return obj, warnings, nil
}

var _ = json.Marshal
