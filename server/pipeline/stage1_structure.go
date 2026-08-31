package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"glam/server/llm"
	scenario_pkg "glam/server/scenario"
	"glam/server/world"
)

func maxStageRetries() int {
	raw := os.Getenv("PIPELINE_MAX_STAGE_RETRIES")
	if raw == "" {
		raw = os.Getenv("PIPELINE_MAX_RETRIES")
	}
	if raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 0 {
			return v
		}
	}
	return 2
}

func GenerateStructure(ctx context.Context, llmClient *llm.OpenRouterClient, deps Deps, topic, ageGroup, templateHint string) (StructureDraft, error) {
	retries := maxStageRetries()
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		draft, err := generateStructureOnce(ctx, llmClient, deps, topic, ageGroup, templateHint)
		if err == nil {
			return draft, nil
		}
		lastErr = err
		if attempt < retries {
			continue
		}
	}
	return StructureDraft{}, fmt.Errorf("stage1 structure failed after %d retries: %w", retries+1, lastErr)
}

func generateStructureOnce(ctx context.Context, llmClient *llm.OpenRouterClient, deps Deps, topic, ageGroup, templateHint string) (StructureDraft, error) {
	layoutSection := llm.BuildLayoutSection()
	tmplHint := ""
	if strings.TrimSpace(templateHint) != "" {
		tmplHint = fmt.Sprintf("Template hint: %s (you may override if topic fits another template better).\n", strings.TrimSpace(templateHint))
	}
	systemPrompt := fmt.Sprintf(`You generate world structure for a GLAM learning scenario.
%s
%s
RULES:
- template must be one of: town, forest, desert, school
- size: cols 8-30, rows 8-20
- entities: 3-8 items, each with id (pattern ^[a-z0-9][a-z0-9_-]*$ unique), kind (character|building), position (x,y inside world.size), plot (optional, must match template: town->plot_*, forest->clearing_*, desert/school omit), role (mission_start|task_building|mission_end|flavor_only)
- At least one mission_start and one mission_end if you have 3+ entities.
- Positions must be inside buildable plots/clearings for town/forest.
- Output JSON only, no markdown.

Topic: %s
Age group: %s
`, tmplHint, layoutSection, topic, ageGroup)

	schema := []byte(`{"type":"object","additionalProperties":false,"required":["template","size","entities"],"properties":{"template":{"type":"string","enum":["town","forest","desert","school"]},"size":{"type":"object","additionalProperties":false,"required":["cols","rows"],"properties":{"cols":{"type":"integer","minimum":8,"maximum":30},"rows":{"type":"integer","minimum":8,"maximum":20}}},"entities":{"type":"array","minItems":1,"maxItems":8,"items":{"type":"object","additionalProperties":false,"required":["id","kind","position","role"],"properties":{"id":{"type":"string","pattern":"^[a-z0-9][a-z0-9_-]*$","maxLength":64},"kind":{"type":"string","enum":["character","building"]},"position":{"type":"object","additionalProperties":false,"required":["x","y"],"properties":{"x":{"type":"integer","minimum":0,"maximum":30},"y":{"type":"integer","minimum":0,"maximum":30}}},"plot":{"type":"string","pattern":"^[a-z0-9][a-z0-9_-]*$"},"role":{"type":"string","enum":["mission_start","task_building","mission_end","flavor_only"]}}}}}`)

	prompt := systemPrompt + "\n\nReturn JSON matching the schema above."

	raw, err := callGenerate(ctx, llmClient, systemPrompt, prompt, schema)
	if err != nil {
		return StructureDraft{}, err
	}
	var draft StructureDraft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return StructureDraft{}, fmt.Errorf("parse structure draft: %w raw=%s", err, raw)
	}
	if err := validateStructure(draft); err != nil {
		return StructureDraft{}, fmt.Errorf("structure validation: %w raw=%s", err, raw)
	}
	return draft, nil
}

func callGenerate(ctx context.Context, client *llm.OpenRouterClient, systemPrompt, userPrompt string, _ []byte) (string, error) {
	// We use the existing Generate but need custom prompts. Instead construct a
	// direct call via GenerateWithTools-like logic using raw messages?
	// Simpler: use llm.Generate with a synthetic schema/registry bypass by
	// calling client.Generate with our prompt — but Generate builds its own
	// system prompt from schema+registry. For stage isolation we want narrow
	// prompts, so we use a low-level helper.
	// For now, emulate by calling GenerateWithTools with no tools and a single
	// system+user message, then extract content.
	msgs := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	res, err := client.GenerateWithTools(ctx, msgs, nil)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(res.Content)
	if text == "" {
		return "", fmt.Errorf("empty LLM response")
	}
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
		text = strings.TrimSpace(strings.Join(lines, "\n"))
		text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "json"))
	}
	var js json.RawMessage
	if err := json.Unmarshal([]byte(text), &js); err != nil {
		start := strings.Index(text, "{")
		end := strings.LastIndex(text, "}")
		if start >= 0 && end > start {
			cand := text[start : end+1]
			if json.Unmarshal([]byte(cand), &js) == nil {
				return cand, nil
			}
		}
		return "", fmt.Errorf("LLM did not return valid JSON: %v raw=%s", err, text)
	}
	return string(js), nil
}

func validateStructure(d StructureDraft) error {
	if d.Template != "town" && d.Template != "forest" && d.Template != "desert" && d.Template != "school" {
		return fmt.Errorf("invalid template %q", d.Template)
	}
	if d.Size.Cols < world.WorldColsMin || d.Size.Cols > world.WorldColsMax {
		return fmt.Errorf("cols %d out of bounds", d.Size.Cols)
	}
	if d.Size.Rows < world.WorldRowsMin || d.Size.Rows > world.WorldRowsMax {
		return fmt.Errorf("rows %d out of bounds", d.Size.Rows)
	}
	seen := map[string]bool{}
	for _, e := range d.Entities {
		if seen[e.ID] {
			return fmt.Errorf("duplicate id %q", e.ID)
		}
		seen[e.ID] = true
		if e.Kind != "character" && e.Kind != "building" {
			return fmt.Errorf("entity %q invalid kind %q", e.ID, e.Kind)
		}
		if e.Position.X < 0 || e.Position.X >= d.Size.Cols || e.Position.Y < 0 || e.Position.Y >= d.Size.Rows {
			return fmt.Errorf("entity %q position out of bounds", e.ID)
		}
		if e.Role != "mission_start" && e.Role != "task_building" && e.Role != "mission_end" && e.Role != "flavor_only" {
			return fmt.Errorf("entity %q invalid role %q", e.ID, e.Role)
		}
	}
	layout := world.GetLayout(d.Template, d.Size.Cols, d.Size.Rows)
	if layout != nil && len(layout.Plots) > 0 {
		for _, e := range d.Entities {
			if !scenario_pkg.PositionInAnyPlot(scenario_pkg.Position{X: e.Position.X, Y: e.Position.Y}, layout.Plots) {
				// Allow but warn? Strict: require inside plot for town/forest
				return fmt.Errorf("entity %q at (%d,%d) not inside any plot/clearing for template %q", e.ID, e.Position.X, e.Position.Y, d.Template)
			}
			if e.Plot != nil && *e.Plot != "" {
				found := false
				inside := false
				for _, p := range layout.Plots {
					if p.ID == *e.Plot {
						found = true
						if scenario_pkg.PositionInPlot(scenario_pkg.Position{X: e.Position.X, Y: e.Position.Y}, p) {
							inside = true
						}
						break
					}
				}
				if !found {
					return fmt.Errorf("entity %q plot %q not found for template %q", e.ID, *e.Plot, d.Template)
				}
				if !inside {
					return fmt.Errorf("entity %q position not inside claimed plot %q", e.ID, *e.Plot)
				}
			}
		}
	} else {
		for _, e := range d.Entities {
			if e.Plot != nil && *e.Plot != "" {
				return fmt.Errorf("entity %q has plot %q but template %q has no plots", e.ID, *e.Plot, d.Template)
			}
		}
	}
	return nil
}
