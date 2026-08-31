package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	scenario_pkg "glam/server/scenario"
	"glam/server/world"
)

func Assemble(structure StructureDraft, flavor FlavorDraft, interactions InteractionDraft, ageStats *scenario_pkg.InitialStats) (map[string]interface{}, error) {
	layout := world.GetLayout(structure.Template, structure.Size.Cols, structure.Size.Rows)
	if layout == nil {
		return nil, fmt.Errorf("invalid layout for template %q size %v", structure.Template, structure.Size)
	}
	spawn := layout.Spawn
	// Ensure spawn walkable; if not, use helper (world ensures walkable, but keep fallback)
	if len(layout.Tilemap) > 0 && layout.Tilemap != nil {
		// layout.Spawn already walkable per world.GetLayout, but keep as is
	}

	scenarioID := slugify(flavor.Title) + fmt.Sprintf("-%d", time.Now().Unix()%100000)
	title := strings.TrimSpace(flavor.Title)
	if title == "" {
		title = "Learning Adventure"
	}

	var characters []map[string]interface{}
	var buildings []map[string]interface{}
	var objectList []map[string]interface{}

	for _, e := range structure.Entities {
		fl := flavor.Names[e.ID]
		interRaw, ok := interactions[e.ID]
		var inter interface{}
		if ok && len(interRaw) > 0 {
			var tmp interface{}
			if json.Unmarshal(interRaw, &tmp) == nil {
				inter = tmp
			}
		}
		if inter == nil {
			inter = map[string]interface{}{"type": "dialogue", "text": fmt.Sprintf("Hello from %s!", fl.Name)}
		}

		entry := map[string]interface{}{
			"id":       e.ID,
			"position": map[string]int{"x": e.Position.X, "y": e.Position.Y},
		}
		if e.Plot != nil && *e.Plot != "" {
			entry["plot"] = *e.Plot
		}
		if inter != nil {
			entry["interaction"] = inter
		}

		switch e.Kind {
		case "character":
			entry["name"] = fl.Name
			if fl.Profession != nil && *fl.Profession != "" {
				entry["profession"] = *fl.Profession
			}
			if fl.TypeAssetID != nil && *fl.TypeAssetID != "" {
				entry["appearance"] = map[string]interface{}{"spriteId": *fl.TypeAssetID}
			}
			characters = append(characters, entry)
		case "building":
			typeAssetID := "shop_small"
			if fl.TypeAssetID != nil && *fl.TypeAssetID != "" {
				typeAssetID = *fl.TypeAssetID
			}
			entry["typeAssetId"] = typeAssetID
			buildings = append(buildings, entry)
		default:
			// object fallback (should not happen with current Kind enum, but handle)
			assetID := "object_chest"
			if fl.TypeAssetID != nil && *fl.TypeAssetID != "" {
				assetID = *fl.TypeAssetID
			}
			entry["assetId"] = assetID
			objectList = append(objectList, entry)
		}
	}

	if characters == nil {
		characters = []map[string]interface{}{}
	}
	if buildings == nil {
		buildings = []map[string]interface{}{}
	}
	if objectList == nil {
		objectList = []map[string]interface{}{}
	}

	missions := buildMissions(structure, flavor, interactions)

	obj := map[string]interface{}{
		"id":      scenarioID,
		"title":   title,
		"version": "1.0",
		"world": map[string]interface{}{
			"template": structure.Template,
			"size":     map[string]int{"cols": structure.Size.Cols, "rows": structure.Size.Rows},
			"spawn":    map[string]int{"x": spawn.X, "y": spawn.Y},
		},
		"characters": characters,
		"buildings":  buildings,
		"objects":    objectList,
		"missions":   missions,
	}
	if ageStats != nil {
		stats := map[string]interface{}{}
		if ageStats.Coins != nil {
			stats["coins"] = *ageStats.Coins
		}
		if ageStats.Lives != nil {
			stats["lives"] = *ageStats.Lives
		}
		if ageStats.Score != nil {
			stats["score"] = *ageStats.Score
		}
		if len(stats) > 0 {
			obj["initialStats"] = stats
		}
	}
	return obj, nil
}

func buildMissions(structure StructureDraft, flavor FlavorDraft, interactions InteractionDraft) []map[string]interface{} {
	var missions []map[string]interface{}
	// Collect by role
	var starts, tasks, ends []EntityDraft
	for _, e := range structure.Entities {
		switch e.Role {
		case "mission_start":
			starts = append(starts, e)
		case "task_building":
			tasks = append(tasks, e)
		case "mission_end":
			ends = append(ends, e)
		}
	}
	mkDone := false

	for i, e := range starts {
		name := flavor.Names[e.ID].Name
		missions = append(missions, map[string]interface{}{
			"id":          fmt.Sprintf("mission_start_%d", i+1),
			"title":       fmt.Sprintf("Meet %s", name),
			"description": fmt.Sprintf("Talk to %s to begin.", name),
			"trigger":     map[string]interface{}{"entityId": e.ID},
			"done":        mkDone,
		})
	}
	for i, e := range tasks {
		name := flavor.Names[e.ID].Name
		m := map[string]interface{}{
			"id":          fmt.Sprintf("mission_task_%d", i+1),
			"title":       fmt.Sprintf("Learn from %s", name),
			"description": fmt.Sprintf("Complete the activity at %s.", name),
			"trigger":     map[string]interface{}{"entityId": e.ID},
			"done":        mkDone,
		}
		if raw, ok := interactions[e.ID]; ok {
			var probe struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(raw, &probe) == nil {
				if probe.Type == "math" || probe.Type == "shop" {
					check := true
					m["checkAtEnd"] = check
					m["requiredStat"] = map[string]interface{}{"stat": "coins", "operator": ">=", "target": 5}
				}
			}
		}
		missions = append(missions, m)
	}
	for i, e := range ends {
		name := flavor.Names[e.ID].Name
		missions = append(missions, map[string]interface{}{
			"id":          fmt.Sprintf("mission_end_%d", i+1),
			"title":       fmt.Sprintf("Finish with %s", name),
			"description": fmt.Sprintf("Return to %s to complete your adventure.", name),
			"trigger":     map[string]interface{}{"entityId": e.ID},
			"done":        mkDone,
		})
	}
	if len(missions) == 0 {
		// fallback: at least one mission from first entity
		if len(structure.Entities) > 0 {
			e := structure.Entities[0]
			missions = append(missions, map[string]interface{}{
				"id":          "mission_1",
				"title":       "Explore",
				"description": "Explore the world and complete activities.",
				"trigger":     map[string]interface{}{"entityId": e.ID},
				"done":        mkDone,
			})
		}
	}
	return missions
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	out := b.String()
	out = strings.Trim(out, "-_")
	if out == "" {
		out = "scenario"
	}
	if len(out) > 30 {
		out = out[:30]
	}
	return out
}
