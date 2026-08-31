package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"glam/server/scenario"
	"glam/server/world"
)

const (
	promptSchemaTruncateCap   = 20000
	promptRegistryTruncateCap = 20000
)

// promptFactualNote: factual sections (schema, registry IDs, layout) are generated from canonical
// schema/scenario.schema.json and schema/asset-registry.json and world layout getters;
// explanatory prose below is manual but values (bounds, types, caps) derive from shared constants.

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
	activityTypesList := strings.Join(scenario.InteractionTypes, ", ")

	schemaStr := string(schemaJSON)
	if len(schemaStr) > promptSchemaTruncateCap {
		schemaStr = schemaStr[:promptSchemaTruncateCap] + "\n...[truncated]"
	}
	registryStr := string(registryJSON)
	if len(registryStr) > promptRegistryTruncateCap {
		registryStr = registryStr[:promptRegistryTruncateCap] + "\n...[truncated]"
	}

	layoutSection := buildLayoutSection()

	return fmt.Sprintf(`You are GLAM scenario generator. Your output will be validated by strict JSON Schema (additionalProperties:false at EVERY level) — ANY property not in the schema will be REJECTED with 400. You must output ONLY valid JSON that validates exactly. No extra keys.

%s

RULES (strict, schema-enforced — additionalProperties:false EVERYWHERE):
- Never generate executable code.
- Only use asset IDs from registry: %s
  - buildings[].typeAssetId must be one of the building asset IDs.
  - objects[].assetId must be one of the object/prop/tile asset IDs.
  - characters[].appearance.spriteId should be one of the character asset IDs if present.
- interaction.type MUST be exactly one of: %s. Use EXACT field names per schema — do NOT invent contentId/dialogueId/outcomes/quizId/shopId/kind/objectives/rewards etc.
  - dialogue: {type:"dialogue", text, speaker?}
  - mcq: {type:"mcq", question, options[{text,correct,explanation?}], allowRetry?}
  - math: {type:"math", question, answer, tolerance?, hint?}
  - shop: {type:"shop", items[{name,price,icon?}], currency?}
  - information: {type:"information", content, title?, image?}
- Positions must be within world.size bounds: 0 <= x < cols, 0 <= y < rows. Spawn must be inside bounds.
- world.size: cols %d-%d, rows %d-%d. world.template must be one of: town, forest, desert, school.
- ids must match pattern ^[a-z0-9][a-z0-9_-]*$ and be unique across characters, buildings, objects, missions.
- Never include forbidden fields: code, script, component, bundle at any level.
- initialStats — ALLOWED KEYS ONLY: coins (0-1000000), lives (0-99), score (0-1000000). Do NOT invent wisdom, xp, health, knowledge, experience, mana etc. Omit initialStats entirely if not needed — engine defaults to coins:40, lives:3, score:0. Example valid: {"coins":50,"lives":3,"score":0}. Invalid (REJECTED): {"wisdom":10} or {"coins":40,"xp":5}.
- missions[] — ALLOWED KEYS ONLY: id, title, description, trigger, checkAtEnd, requiredStat, done. Do NOT invent reward, goal, objectives, outcome, xp, coinsReward, kind, status, completed, rewardCoins etc. Each mission example valid:
    {"id":"talk_teacher","title":"Get your budget","description":"Talk to Ms. Rao.","trigger":{"entityId":"ms_rao"},"done":false}
    {"id":"save_coins","title":"Save wisely","description":"End with >=10 coins.","trigger":{"entityId":"town_bank"},"checkAtEnd":true,"requiredStat":{"stat":"coins","operator":">=","target":10},"done":false}
  Valid trigger keys ONLY: entityId, interactionId, auto. Valid requiredStat keys ONLY: stat, operator (one of >= > <= < = == !=), target (number).
- ANY extra property at ANY path (e.g. /initialStats/wisdom, /missions/0/reward, /missions/0/goal) causes immediate rejection (additionalProperties:false). When in doubt, OMIT optional fields rather than invent them.
- missions[].trigger.entityId must refer to an existing entity id if present.
- CRITICAL: plot field is TEMPLATE-LOCKED. This is the #1 validation failure cause.
  - If world.template=="town": you may ONLY use plot_1..plot_6 (town plots). NEVER use clearing_*.
  - If world.template=="forest": you may ONLY use clearing_1..clearing_4 (forest clearings). NEVER use plot_*.
  - If world.template=="desert" or "school": there are NO plots — you MUST omit the plot field entirely (omit it, do not set it to null). Any plot value will be rejected.
  - Cross-template plot IDs (e.g. clearing_2 inside town, or plot_1 inside forest) are INVALID and will cause immediate rejection. When in doubt, omit plot — position being inside SOME buildable plot is enough, plot is optional.
- Return pure JSON only — no prose, no markdown fences, no explanation. Just the JSON object that matches the schema exactly.

JSON Schema:
%s

Asset Registry (valid IDs):
%s

Valid asset IDs list: [%s]
Valid activity types: [%s]
`, layoutSection, idsList, activityTypesList, world.WorldColsMin, world.WorldColsMax, world.WorldRowsMin, world.WorldRowsMax, schemaStr, registryStr, idsList, activityTypesList)
}

func buildLayoutSection() string {
	town := world.GetLayout("town", 15, 12)
	forest := world.GetLayout("forest", 15, 12)
	var b strings.Builder
	b.WriteString("WORLD LAYOUTS (fixed, you MUST respect):\n")
	if town != nil {
		rx1 := 15 / 3
		rx2 := (2 * 15) / 3
		ry1 := 12 / 3
		ry2 := (2 * 12) / 3
		b.WriteString(fmt.Sprintf("TOWN (15x12 example, roads are path tiles walkable, do NOT place buildings on roads):\n"))
		b.WriteString(fmt.Sprintf("  Roads: horizontal y=%d & y=%d across cols, vertical x=%d & x=%d across rows (1 tile thick path) — example 15x12: y=4 & y=8, x=5 & x=9 (computed x=%d & x=%d)\n", ry1, ry2, rx1, rx2, rx1, rx2))
		b.WriteString("  Open plots (buildable, place buildings/characters/objects INSIDE these rectangles) — ONLY for template=town:\n")
		for _, p := range town.Plots {
			b.WriteString(fmt.Sprintf("    %s: x=%d,y=%d,w=%d,h=%d (%s)\n", p.ID, p.X, p.Y, p.W, p.H, p.Name))
		}
		b.WriteString("  Note: grass is walkable, path is walkable, do NOT place on roads if you want inside plots.\n")
		b.WriteString("  TOWN RULE: when world.template=\"town\", plot field MUST be one of plot_1..plot_6 OR omitted. NEVER use clearing_* in town.\n")
	}
	if forest != nil {
		b.WriteString("FOREST (15x12, trees solid blocked, grass walkable, clearings buildable) — ONLY for template=forest:\n")
		b.WriteString("  Trees: border all edges (x==0||y==0||x==cols-1||y==rows-1 => tree) + scattered interior deterministic (13% density) skipped inside clearings\n")
		b.WriteString("  Clearings (open, buildable) — ONLY for template=forest:\n")
		for _, p := range forest.Plots {
			b.WriteString(fmt.Sprintf("    %s: x=%d,y=%d,w=%d,h=%d (%s)\n", p.ID, p.X, p.Y, p.W, p.H, p.Name))
		}
		b.WriteString("  FOREST RULE: when world.template=\"forest\", plot field MUST be one of clearing_1..clearing_4 OR omitted. NEVER use plot_* in forest.\n")
	}
	b.WriteString("DESERT/SCHOOL: no roads/trees, no fixed plots (entire interior grass is walkable). When world.template=\"desert\" or \"school\", OMIT plot field entirely.\n")
	b.WriteString("RULE: Every entity position must be inside a plot/clearing (town/forest) and not on tree/water/road edge. Spawn must be walkable (grass/path). For desert/school any walkable tile is fine.\n")
	b.WriteString("  For generic sizes: town roads at cols/3 and 2*cols/3 vertical, rows/3 and 2*rows/3 horizontal; forest border trees + clearings proportional (see forest.go for exact formula) — scale plots proportionally, do NOT hardcode 15x12 coords for other sizes.\n")
	b.WriteString("  Example town (ONLY when template=town): to place hospital at plot_1, use {\"id\":\"hosp1\",\"typeAssetId\":\"hospital\",\"position\":{\"x\":2,\"y\":2},\"plot\":\"plot_1\"} where (2,2) lies inside plot_1 bounds — you MUST include plot field matching the plot you target.\n")
	b.WriteString("  Example forest (ONLY when template=forest): to place ranger at clearing_2, use {\"id\":\"ranger1\",\"name\":\"Ranger\",\"position\":{\"x\":11,\"y\":3},\"plot\":\"clearing_2\"} inside clearing_2 (10,2,4,3) — do NOT use this example for town.\n")
	b.WriteString("  Cross-template example (INVALID, will be rejected): town with clearing_2, or forest with plot_1 — NEVER do this.\n")
	b.WriteString("  You may omit plot field but position must still be inside SOME plot/clearing for town/forest — omission is safer than wrong plot. Plot field is optional.\n")
	b.WriteString("  Spawn example: {\"x\":7,\"y\":9} is walkable grass not on road/tree.\n")
	return b.String()
}
