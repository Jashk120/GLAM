package scenario

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ValidActivityTypes is the allowed interaction types.
var ValidActivityTypes = map[string]bool{
	"dialogue":    true,
	"mcq":         true,
	"math":        true,
	"shop":        true,
	"information": true,
}

var forbiddenFields = []string{"code", "script", "component", "bundle"}

// ValidateScenario validates data in order:
// 1 JSON parse -> 2 Schema -> 3 asset-ID -> 4 activity-ID -> 5 reference/position -> 6 forbidden fields
func ValidateScenario(data []byte, schemaPath string, registryPath string) (bool, []string, error) {
	var errs []string

	// Step 1: JSON parsing
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false, []string{fmt.Sprintf("JSON parse error: %v", err)}, nil
	}
	var genericMap map[string]interface{}
	if err := json.Unmarshal(data, &genericMap); err != nil {
		return false, []string{fmt.Sprintf("JSON parse error: %v", err)}, nil
	}

	// Step 6 (forbidden fields) check early but reported after schema - however spec says order is schema -> asset -> activity -> reference
	// We'll collect forbidden field errors separately.
	forbiddenErrs := checkForbiddenFields(genericMap, "")

	// Step 2: JSON Schema validation
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		return false, nil, fmt.Errorf("compile schema: %w", err)
	}
	// Need to validate as generic interface
	var doc interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return false, []string{fmt.Sprintf("JSON parse error: %v", err)}, nil
	}
	if err := schema.Validate(doc); err != nil {
		// Collect schema errors
		if ve, ok := err.(*jsonschema.ValidationError); ok {
			for _, ce := range flattenValidationErrors(ve) {
				errs = append(errs, ce)
			}
		} else {
			errs = append(errs, err.Error())
		}
	}

	// If schema validation failed, still continue to asset/activity checks for detailed errors?
	// But we return them all.

	// Append forbidden field errors
	errs = append(errs, forbiddenErrs...)

	// Parse into typed struct for deeper checks
	var sc Scenario
	if err := json.Unmarshal(data, &sc); err != nil {
		// If typed parse fails, return schema errors already collected
		// Add typed parse error if not already covered
		if len(errs) == 0 {
			errs = append(errs, fmt.Sprintf("typed parse error: %v", err))
		}
		return false, errs, nil
	}

	// Load registry for asset checks
	registryIDs := map[string]bool{}
	if registryPath != "" {
		regMap, _, err := LoadRegistry(registryPath)
		if err == nil {
			for id := range regMap {
				registryIDs[id] = true
			}
		} else {
			errs = append(errs, fmt.Sprintf("registry load error: %v", err))
		}
	}

	// Step 3: asset-ID validation
	for _, b := range sc.Buildings {
		if len(registryIDs) > 0 && !registryIDs[b.TypeAssetID] {
			errs = append(errs, fmt.Sprintf("building %q: typeAssetId %q not found in registry", b.ID, b.TypeAssetID))
		}
	}
	for _, o := range sc.Objects {
		if len(registryIDs) > 0 && !registryIDs[o.AssetID] {
			errs = append(errs, fmt.Sprintf("object %q: assetId %q not found in registry", o.ID, o.AssetID))
		}
	}

	// Step 4: activity-ID validation (interaction.type must be one of 5)
	allInteractions := collectInteractions(&sc)
	for _, entry := range allInteractions {
		if !ValidActivityTypes[entry.Type] {
			errs = append(errs, fmt.Sprintf("entity %q: interaction type %q is not valid (must be one of dialogue, mcq, math, shop, information)", entry.EntityID, entry.Type))
		}
	}

	// Step 5: reference/position validation
	cols := sc.World.Size.Cols
	rows := sc.World.Size.Rows

	// Spawn inside bounds
	if sc.World.Spawn.X < 0 || sc.World.Spawn.X >= cols {
		errs = append(errs, fmt.Sprintf("world.spawn.x %d out of bounds [0,%d)", sc.World.Spawn.X, cols))
	}
	if sc.World.Spawn.Y < 0 || sc.World.Spawn.Y >= rows {
		errs = append(errs, fmt.Sprintf("world.spawn.y %d out of bounds [0,%d)", sc.World.Spawn.Y, rows))
	}

	// Positions within bounds
	checkPosition := func(label string, pos Position) {
		if pos.X < 0 || pos.X >= cols {
			errs = append(errs, fmt.Sprintf("%s position x=%d out of bounds [0,%d)", label, pos.X, cols))
		}
		if pos.Y < 0 || pos.Y >= rows {
			errs = append(errs, fmt.Sprintf("%s position y=%d out of bounds [0,%d)", label, pos.Y, rows))
		}
	}
	for _, c := range sc.Characters {
		checkPosition(fmt.Sprintf("character %q", c.ID), c.Position)
	}
	for _, b := range sc.Buildings {
		checkPosition(fmt.Sprintf("building %q", b.ID), b.Position)
		if b.Width != nil {
			if *b.Width < 1 || *b.Width > 30 {
				errs = append(errs, fmt.Sprintf("building %q: width %d out of range", b.ID, *b.Width))
			} else if b.Position.X+*b.Width > cols {
				errs = append(errs, fmt.Sprintf("building %q: position x+width %d exceeds world cols %d", b.ID, b.Position.X+*b.Width, cols))
			}
		}
		if b.Height != nil {
			if *b.Height < 1 || *b.Height > 20 {
				errs = append(errs, fmt.Sprintf("building %q: height %d out of range", b.ID, *b.Height))
			} else if b.Position.Y+*b.Height > rows {
				errs = append(errs, fmt.Sprintf("building %q: position y+height %d exceeds world rows %d", b.ID, b.Position.Y+*b.Height, rows))
			}
		}
	}
	for _, o := range sc.Objects {
		checkPosition(fmt.Sprintf("object %q", o.ID), o.Position)
	}

	// Duplicate entity IDs check
	seen := map[string]string{}
	addID := func(id, kind string) {
		if prev, ok := seen[id]; ok {
			errs = append(errs, fmt.Sprintf("duplicate entity id %q (already used by %s, now %s)", id, prev, kind))
		} else {
			seen[id] = kind
		}
	}
	for _, c := range sc.Characters {
		addID(c.ID, "character")
	}
	for _, b := range sc.Buildings {
		addID(b.ID, "building")
	}
	for _, o := range sc.Objects {
		addID(o.ID, "object")
	}
	for _, m := range sc.Missions {
		addID(m.ID, "mission")
	}

	// Mission trigger IDs refer to existing entities
	for _, m := range sc.Missions {
		if m.Trigger != nil && m.Trigger.EntityID != nil {
			eid := *m.Trigger.EntityID
			if _, ok := seen[eid]; !ok {
				errs = append(errs, fmt.Sprintf("mission %q: trigger entityId %q not found", m.ID, eid))
			} else {
				// ensure it refers to non-mission entity
				kind := seen[eid]
				if kind == "mission" {
					errs = append(errs, fmt.Sprintf("mission %q: trigger entityId %q refers to a mission, expected character/building/object", m.ID, eid))
				}
			}
		}
	}

	if len(errs) > 0 {
		return false, errs, nil
	}
	return true, nil, nil
}

type interactionEntry struct {
	EntityID string
	Type     string
}

func collectInteractions(sc *Scenario) []interactionEntry {
	var out []interactionEntry
	for _, c := range sc.Characters {
		if c.Interaction != nil {
			out = append(out, interactionEntry{EntityID: c.ID, Type: c.Interaction.Type})
		}
	}
	for _, b := range sc.Buildings {
		if b.Interaction != nil {
			out = append(out, interactionEntry{EntityID: b.ID, Type: b.Interaction.Type})
		}
	}
	for _, o := range sc.Objects {
		if o.Interaction != nil {
			out = append(out, interactionEntry{EntityID: o.ID, Type: o.Interaction.Type})
		}
	}
	return out
}

func checkForbiddenFields(m map[string]interface{}, prefix string) []string {
	var errs []string
	for k, v := range m {
		for _, f := range forbiddenFields {
			if k == f {
				label := k
				if prefix != "" {
					label = prefix + "." + k
				}
				errs = append(errs, fmt.Sprintf("forbidden field %q not allowed", label))
			}
		}
		// recurse into objects and arrays
		if child, ok := v.(map[string]interface{}); ok {
			sub := k
			if prefix != "" {
				sub = prefix + "." + k
			}
			errs = append(errs, checkForbiddenFields(child, sub)...)
		} else if arr, ok := v.([]interface{}); ok {
			for i, elem := range arr {
				if child, ok := elem.(map[string]interface{}); ok {
					sub := fmt.Sprintf("%s[%d]", k, i)
					if prefix != "" {
						sub = prefix + "." + sub
					}
					errs = append(errs, checkForbiddenFields(child, sub)...)
				}
			}
		}
	}
	return errs
}

func flattenValidationErrors(ve *jsonschema.ValidationError) []string {
	var out []string
	if len(ve.Causes) == 0 {
		msg := ve.Message
		if msg == "" {
			msg = ve.Error()
		}
		if ve.InstanceLocation != "" && ve.InstanceLocation != "/" {
			msg = fmt.Sprintf("%s: %s", ve.InstanceLocation, msg)
		}
		out = append(out, strings.TrimSpace(msg))
		return out
	}
	for _, c := range ve.Causes {
		out = append(out, flattenValidationErrors(c)...)
	}
	return out
}

// ValidateAndParse validates and returns typed Scenario if valid.
func ValidateAndParse(data []byte, schemaPath string, registryPath string) (*Scenario, []string, error) {
	ok, errs, err := ValidateScenario(data, schemaPath, registryPath)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, errs, nil
	}
	var sc Scenario
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, []string{fmt.Sprintf("parse after validation: %v", err)}, nil
	}
	return &sc, nil, nil
}
