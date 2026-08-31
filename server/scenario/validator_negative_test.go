package scenario

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testPaths() (string, string) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	schema := filepath.Join(dir, "..", "..", "schema/scenario.schema.json")
	registry := filepath.Join(dir, "..", "..", "schema/asset-registry.json")
	return schema, registry
}

func validMap() map[string]interface{} {
	return map[string]interface{}{
		"id":      "test-scenario",
		"title":   "Test Scenario",
		"version": "1.0",
		"world": map[string]interface{}{
			"template": "desert",
			"spawn":    map[string]interface{}{"x": 2, "y": 2},
			"size":     map[string]interface{}{"cols": 15, "rows": 12},
		},
		"characters": []interface{}{
			map[string]interface{}{"id": "char_a", "name": "Alice", "position": map[string]interface{}{"x": 1, "y": 1}},
		},
		"buildings": []interface{}{
			map[string]interface{}{"id": "bld_a", "typeAssetId": "shop_small", "position": map[string]interface{}{"x": 3, "y": 3}},
		},
		"objects": []interface{}{
			map[string]interface{}{"id": "obj_a", "assetId": "object_chest", "position": map[string]interface{}{"x": 4, "y": 4}},
		},
		"missions": []interface{}{
			map[string]interface{}{"id": "mission_a", "title": "Talk", "description": "Talk to Alice", "trigger": map[string]interface{}{"entityId": "char_a"}},
		},
	}
}

func mustBytes(m map[string]interface{}) []byte {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return b
}

func cloneMap(m map[string]interface{}) map[string]interface{} {
	b, _ := json.Marshal(m)
	var out map[string]interface{}
	_ = json.Unmarshal(b, &out)
	return out
}

func TestValidateScenario_Negative(t *testing.T) {
	schema, registry := testPaths()
	t.Run("json_parse_error", func(t *testing.T) {
		ok, details, err := ValidateScenario([]byte(`{ invalid json`), schema, registry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatal("expected invalid for malformed JSON")
		}
		if len(details) == 0 || !strings.Contains(strings.Join(details, " "), "JSON parse") {
			t.Fatalf("expected JSON parse error, got %v", details)
		}
	})
	cases := []struct {
		name     string
		mutate   func(map[string]interface{})
		contains string
	}{
		{name: "missing_id", mutate: func(m map[string]interface{}) { delete(m, "id") }, contains: "id"},
		{name: "missing_title", mutate: func(m map[string]interface{}) { delete(m, "title") }, contains: "title"},
		{name: "missing_world", mutate: func(m map[string]interface{}) { delete(m, "world") }, contains: "world"},
		{name: "invalid_id_uppercase", mutate: func(m map[string]interface{}) { m["id"] = "Bad_ID" }, contains: "pattern"},
		{name: "invalid_id_slash", mutate: func(m map[string]interface{}) { m["id"] = "bad/id" }, contains: "pattern"},
		{name: "invalid_id_empty", mutate: func(m map[string]interface{}) { m["id"] = "" }, contains: "length"},
		{name: "invalid_version_pattern", mutate: func(m map[string]interface{}) { m["version"] = "bad_version" }, contains: "pattern"},
		{name: "invalid_world_template", mutate: func(m map[string]interface{}) { m["world"].(map[string]interface{})["template"] = "ocean" }, contains: "template"},
		{name: "spawn_x_out_of_30", mutate: func(m map[string]interface{}) {
			m["world"].(map[string]interface{})["spawn"] = map[string]interface{}{"x": 31, "y": 2}
		}, contains: "must be <="},
		{name: "size_cols_too_small", mutate: func(m map[string]interface{}) {
			m["world"].(map[string]interface{})["size"] = map[string]interface{}{"cols": 7, "rows": 12}
		}, contains: "must be >="},
		{name: "size_rows_too_large", mutate: func(m map[string]interface{}) {
			m["world"].(map[string]interface{})["size"] = map[string]interface{}{"cols": 15, "rows": 21}
		}, contains: "must be <="},
		{name: "missing_world_size", mutate: func(m map[string]interface{}) { delete(m["world"].(map[string]interface{}), "size") }, contains: "size"},
		{name: "spawn_gte_cols", mutate: func(m map[string]interface{}) {
			w := m["world"].(map[string]interface{})
			w["size"] = map[string]interface{}{"cols": 8, "rows": 8}
			w["spawn"] = map[string]interface{}{"x": 8, "y": 2}
		}, contains: "out of bounds"},
		{name: "asset_building_fake", mutate: func(m map[string]interface{}) {
			m["buildings"].([]interface{})[0].(map[string]interface{})["typeAssetId"] = "fake_building_xyz"
		}, contains: "not found in registry"},
		{name: "asset_object_fake", mutate: func(m map[string]interface{}) {
			m["objects"].([]interface{})[0].(map[string]interface{})["assetId"] = "fake_object_xyz"
		}, contains: "not found in registry"},
		{name: "duplicate_entity_ids", mutate: func(m map[string]interface{}) {
			m["buildings"].([]interface{})[0].(map[string]interface{})["id"] = "char_a"
		}, contains: "duplicate"},
		{name: "mission_trigger_missing_entity", mutate: func(m map[string]interface{}) {
			m["missions"].([]interface{})[0].(map[string]interface{})["trigger"] = map[string]interface{}{"entityId": "ghost_xyz"}
		}, contains: "not found"},
		{name: "mission_trigger_points_to_mission", mutate: func(m map[string]interface{}) {
			m["missions"].([]interface{})[0].(map[string]interface{})["trigger"] = map[string]interface{}{"entityId": "mission_a"}
		}, contains: "refers to a mission"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := cloneMap(validMap())
			tc.mutate(m)
			data := mustBytes(m)
			ok, details, err := ValidateScenario(data, schema, registry)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok {
				t.Fatalf("expected valid==false for %s", tc.name)
			}
			if len(details) == 0 {
				t.Fatalf("expected details for %s", tc.name)
			}
			if !strings.Contains(strings.ToLower(strings.Join(details, " | ")), strings.ToLower(tc.contains)) {
				t.Fatalf("case %s: details %q does not contain %q", tc.name, strings.Join(details, " | "), tc.contains)
			}
		})
	}
}
