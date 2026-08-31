package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"glam/server/llm"
	scenario_pkg "glam/server/scenario"
)

func GenerateFlavor(ctx context.Context, llmClient *llm.OpenRouterClient, deps Deps, draft StructureDraft, topic string) (FlavorDraft, error) {
	registryMap, _, err := scenario_pkg.LoadRegistry(deps.RegistryPath)
	if err != nil {
		// try fallback: use deps.RegistryJSON if LoadRegistry fails
		registryMap = map[string]scenario_pkg.Asset{}
		if len(deps.RegistryJSON) > 0 {
			var assets []scenario_pkg.Asset
			if json.Unmarshal(deps.RegistryJSON, &assets) == nil {
				for _, a := range assets {
					registryMap[a.ID] = a
				}
			}
		}
	}
	var buildingIDs, characterIDs []string
	for _, id := range registryAssetIDs(registryMap, "building") {
		buildingIDs = append(buildingIDs, id)
	}
	for _, id := range registryAssetIDs(registryMap, "character") {
		characterIDs = append(characterIDs, id)
	}

	var entityList strings.Builder
	for _, e := range draft.Entities {
		entityList.WriteString(fmt.Sprintf("- %s (%s, role=%s) at (%d,%d)\n", e.ID, e.Kind, e.Role, e.Position.X, e.Position.Y))
	}

	systemPrompt := fmt.Sprintf(`You generate flavor names for a learning scenario.
Topic: %s
Template: %s
Entities:
%s
Building asset IDs allowed: %s
Character sprite IDs allowed: %s

Rules:
- For each entity id, provide name (1-40 chars), profession if character, typeAssetId if building (must be from building list)
- Title: 1-80 chars, describes the scenario.
- Output JSON only: {"title":"...", "names":{"<id>":{"name":"...","profession":"...","typeAssetId":"..."}}}
`, topic, draft.Template, entityList.String(), strings.Join(buildingIDs, ", "), strings.Join(characterIDs, ", "))

	schema := []byte(`{"type":"object","additionalProperties":false,"required":["title","names"],"properties":{"title":{"type":"string","minLength":1,"maxLength":100},"names":{"type":"object","additionalProperties":{"type":"object","additionalProperties":false,"properties":{"name":{"type":"string","minLength":1,"maxLength":80},"profession":{"type":"string","minLength":1,"maxLength":64},"typeAssetId":{"type":"string","pattern":"^[a-z0-9][a-z0-9_-]*$"}}}}}}`)

	userPrompt := systemPrompt + "\nReturn JSON matching schema."

	raw, err := callGenerate(ctx, llmClient, systemPrompt, userPrompt, schema)
	if err != nil {
		return FlavorDraft{}, err
	}
	var out FlavorDraft
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return FlavorDraft{}, fmt.Errorf("parse flavor draft: %w raw=%s", err, raw)
	}
	if strings.TrimSpace(out.Title) == "" {
		return FlavorDraft{}, fmt.Errorf("flavor title empty")
	}
	for _, e := range draft.Entities {
		fl, ok := out.Names[e.ID]
		if !ok {
			return FlavorDraft{}, fmt.Errorf("missing flavor for entity %q", e.ID)
		}
		if strings.TrimSpace(fl.Name) == "" {
			return FlavorDraft{}, fmt.Errorf("empty name for %q", e.ID)
		}
		if e.Kind == "building" && fl.TypeAssetID != nil && *fl.TypeAssetID != "" {
			if _, ok := registryMap[*fl.TypeAssetID]; !ok {
				return FlavorDraft{}, fmt.Errorf("building %q typeAssetId %q not in registry", e.ID, *fl.TypeAssetID)
			}
		}
	}
	return out, nil
}

func registryAssetIDs(m map[string]scenario_pkg.Asset, typ string) []string {
	var out []string
	for id, a := range m {
		if a.Type == typ {
			out = append(out, id)
		}
	}
	return out
}
