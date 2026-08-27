package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildSystemPrompt creates the system prompt that enforces GLAM rules.
// schemaJSON and registryJSON are the raw bytes (truncated if needed).
func BuildSystemPrompt(schemaJSON []byte, registryJSON []byte) string {
	var registryIDs []string
	var assets []map[string]interface{}
	if err := json.Unmarshal(registryJSON, &assets); err == nil {
		for _, a := range assets {
			if id, ok := a["id"].(string); ok {
				registryIDs = append(registryIDs, id)
			}
		}
	}

	idsList := strings.Join(registryIDs, ", ")

	// Truncate schema/registry if too large, but include full for correctness (cap prompt size)
	schemaStr := string(schemaJSON)
	if len(schemaStr) > 8000 {
		schemaStr = schemaStr[:8000] + "\n...[truncated]"
	}
	registryStr := string(registryJSON)
	if len(registryStr) > 8000 {
		registryStr = registryStr[:8000] + "\n...[truncated]"
	}

	return fmt.Sprintf(`You are GLAM scenario generator. You must output ONLY valid JSON matching the provided JSON Schema.

RULES (strict):
- Never generate executable code.
- Only use asset IDs from registry: %s
  - buildings[].typeAssetId must be one of the building asset IDs.
  - objects[].assetId must be one of the object/prop/tile asset IDs.
  - characters[].appearance.spriteId should be one of the character asset IDs if present.
- Only use activity types: dialogue, mcq, math, shop, information for interaction.type.
- Positions must be within world.size bounds: 0 <= x < cols, 0 <= y < rows. Spawn must be inside bounds.
- world.size: cols 8-30, rows 8-20. world.template must be one of: town, forest, desert, school.
- ids must match pattern ^[a-z0-9][a-z0-9_-]*$ and be unique across characters, buildings, objects, missions.
- Never include forbidden fields: code, script, component, bundle at any level.
- missions[].trigger.entityId must refer to an existing entity id if present.
- Return pure JSON only — no prose, no markdown fences, no explanation. Just the JSON object.

JSON Schema:
%s

Asset Registry (valid IDs):
%s

Valid asset IDs list: [%s]
Valid activity types: [dialogue, mcq, math, shop, information]
`, idsList, schemaStr, registryStr, idsList)
}
