package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"glam/server/llm"
	scenario_pkg "glam/server/scenario"
)

func GenerateInteractions(ctx context.Context, llmClient *llm.OpenRouterClient, deps Deps, draft StructureDraft, flavor FlavorDraft, topic, ageGroup string) (InteractionDraft, error) {
	var entityDesc strings.Builder
	for _, e := range draft.Entities {
		name := e.ID
		if f, ok := flavor.Names[e.ID]; ok {
			name = f.Name
		}
		entityDesc.WriteString(fmt.Sprintf("- %s (%s, role=%s, name=%q)\n", e.ID, e.Kind, e.Role, name))
	}

	systemPrompt := fmt.Sprintf(`You generate interactions for each entity in a learning scenario.
Topic: %s
Age group: %s
Template: %s
Entities:
%s

Rules:
- For EACH entity id, produce ONE interaction object keyed by id.
- Valid types: dialogue {type:"dialogue", text (1-1000), speaker?}, mcq {type:"mcq", question, options[2-5]{text,correct,explanation?}, allowRetry?}, math {type:"math", question, answer (number|string), tolerance?, hint?}, shop {type:"shop", currency?, items[1-10]{name,price,icon?}}, information {type:"information", content (1-3000), title?, image?}
- Optional common: cooldown, auto, onCorrect/onWrong {stat?,delta?,toast?}
- Content must suit age group and topic.
- Output JSON object: {"<entityId>": {interaction object}, ...} covering ALL ids.
`, topic, ageGroup, draft.Template, entityDesc.String())

	schema := []byte(`{"type":"object","additionalProperties":{"type":"object"}}`)
	userPrompt := systemPrompt + "\nReturn JSON mapping entity id to interaction object."

	raw, err := callGenerate(ctx, llmClient, systemPrompt, userPrompt, schema)
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("parse interactions raw: %w raw=%s", err, raw)
	}
	// Ensure every entity has an entry; fill missing with fallback dialogue
	result := make(InteractionDraft)
	var failed []string
	for _, e := range draft.Entities {
		rawEntry, ok := m[e.ID]
		if !ok {
			failed = append(failed, e.ID)
			continue
		}
		ok2, details, err := scenario_pkg.ValidateInteractionFragment(rawEntry)
		if err != nil {
			return nil, fmt.Errorf("validate fragment %q: %w", e.ID, err)
		}
		if !ok2 {
			failed = append(failed, e.ID)
			_ = details
			continue
		}
		result[e.ID] = rawEntry
	}

	if len(failed) > 0 {
		repairPrompt := fmt.Sprintf("The following entity ids produced invalid interactions: %s\nErrors: validation failed per #/$defs/interaction. Return ONLY corrected objects for these ids as JSON map {\"<id>\": {interaction}, ...} with valid types (dialogue|mcq|math|shop|information). Keep content age-appropriate.", strings.Join(failed, ", "))
		repairSystem := systemPrompt + "\n\nREPAIR: correct only the failed ids."
		repairRaw, err := callGenerate(ctx, llmClient, repairSystem, repairPrompt, schema)
		if err == nil {
			var repairMap map[string]json.RawMessage
			if json.Unmarshal([]byte(repairRaw), &repairMap) == nil {
				for _, fid := range failed {
					if rv, ok := repairMap[fid]; ok {
						if ok2, _, _ := scenario_pkg.ValidateInteractionFragment(rv); ok2 {
							result[fid] = rv
						}
					}
				}
			}
		}
		// Still missing? Fallback to safe dialogue
		for _, fid := range failed {
			if _, ok := result[fid]; !ok {
				name := fid
				if f, ok := flavor.Names[fid]; ok {
					name = f.Name
				}
				fallback := map[string]interface{}{
					"type": "dialogue",
					"text": fmt.Sprintf("Hello from %s! Let's learn about %s.", name, topic),
				}
				b, _ := json.Marshal(fallback)
				result[fid] = json.RawMessage(b)
			}
		}
	}
	return result, nil
}
