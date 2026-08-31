package scenario

import (
	"strings"
	"testing"
)

func TestValidateScenario_Negative2(t *testing.T) {
	schema, registry := testPaths()
	cases := []struct {
		name     string
		mutate   func(map[string]interface{})
		contains string
	}{
		{name: "forbidden_field_top_level_code", mutate: func(m map[string]interface{}) { m["code"] = "evil" }, contains: "forbidden"},
		{name: "forbidden_field_nested_script", mutate: func(m map[string]interface{}) {
			m["characters"].([]interface{})[0].(map[string]interface{})["script"] = "evil"
		}, contains: "forbidden"},
		{name: "forbidden_field_bundle_nested", mutate: func(m map[string]interface{}) { m["world"].(map[string]interface{})["bundle"] = "evil" }, contains: "forbidden"},
		{name: "forbidden_field_component", mutate: func(m map[string]interface{}) { m["component"] = "evil" }, contains: "forbidden"},
		{name: "additionalProperties_building_extra", mutate: func(m map[string]interface{}) {
			m["buildings"].([]interface{})[0].(map[string]interface{})["extraField"] = "oops"
		}, contains: "additionalProperties"},
		{name: "interaction_type_invalid", mutate: func(m map[string]interface{}) {
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "quiz", "text": "hi"}
		}, contains: "interaction"},
		{name: "mcq_options_too_few", mutate: func(m map[string]interface{}) {
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "mcq", "question": "Q?", "options": []interface{}{map[string]interface{}{"text": "only", "correct": true}}}
		}, contains: "minimum 2"},
		{name: "mcq_options_too_many", mutate: func(m map[string]interface{}) {
			opts := make([]interface{}, 6)
			for i := 0; i < 6; i++ {
				opts[i] = map[string]interface{}{"text": "opt", "correct": i == 0}
			}
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "mcq", "question": "Q?", "options": opts}
		}, contains: "maximum 5"},
		{name: "shop_items_empty", mutate: func(m map[string]interface{}) {
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "shop", "items": []interface{}{}}
		}, contains: "minimum 1"},
		{name: "shop_price_negative", mutate: func(m map[string]interface{}) {
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "shop", "items": []interface{}{map[string]interface{}{"name": "Bread", "price": -5}}}
		}, contains: "must be >="},
		{name: "information_content_empty", mutate: func(m map[string]interface{}) {
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "information", "content": ""}
		}, contains: "length"},
		{name: "tolerance_negative", mutate: func(m map[string]interface{}) {
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "math", "question": "1+1?", "answer": 2, "tolerance": -1}
		}, contains: "must be >="},
		{name: "cooldown_negative", mutate: func(m map[string]interface{}) {
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "dialogue", "text": "hi", "cooldown": -100}
		}, contains: "must be >="},
		{name: "building_width_overflow", mutate: func(m map[string]interface{}) {
			b := m["buildings"].([]interface{})[0].(map[string]interface{})
			b["position"] = map[string]interface{}{"x": 14, "y": 3}
			b["width"] = 2
		}, contains: "exceeds world cols"},
		{name: "regions_maxItems_exceeded", mutate: func(m map[string]interface{}) {
			regs := make([]interface{}, 21)
			for i := 0; i < 21; i++ {
				regs[i] = map[string]interface{}{"id": "r"}
			}
			m["world"].(map[string]interface{})["regions"] = regs
		}, contains: "maximum 20"},
		{name: "shop_currency_empty", mutate: func(m map[string]interface{}) {
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "shop", "currency": "", "items": []interface{}{map[string]interface{}{"name": "x", "price": 1}}}
		}, contains: "length"},
		{name: "shop_items_too_many_11", mutate: func(m map[string]interface{}) {
			items := make([]interface{}, 11)
			for i := 0; i < 11; i++ {
				items[i] = map[string]interface{}{"name": "x", "price": 1}
			}
			m["characters"].([]interface{})[0].(map[string]interface{})["interaction"] = map[string]interface{}{"type": "shop", "items": items}
		}, contains: "maximum 10"},
		{name: "building_height_overflow", mutate: func(m map[string]interface{}) {
			b := m["buildings"].([]interface{})[0].(map[string]interface{})
			b["position"] = map[string]interface{}{"x": 3, "y": 11}
			b["height"] = 2
		}, contains: "exceeds world rows"},
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

func TestValidateScenario_SpriteId_NotChecked_Gap(t *testing.T) {
	schema, registry := testPaths()
	m := validMap()
	m["characters"].([]interface{})[0].(map[string]interface{})["appearance"] = map[string]interface{}{"spriteId": "fake_sprite_xyz"}
	data := mustBytes(m)
	ok, details, err := ValidateScenario(data, schema, registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected invalid for fake spriteId, got valid==true")
	}
	joined := strings.Join(details, " | ")
	if !strings.Contains(strings.ToLower(joined), "not in registry") {
		t.Fatalf("expected 'not in registry' error for spriteId, got %q", joined)
	}
	if !strings.Contains(joined, "fake_sprite_xyz") {
		t.Fatalf("expected spriteId value in error, got %q", joined)
	}
}

func TestValidateScenario_ForestSolidTile(t *testing.T) {
	schema, registry := testPaths()
	m := validMap()
	w := m["world"].(map[string]interface{})
	w["template"] = "forest"
	w["size"] = map[string]interface{}{"cols": 15, "rows": 12}
	m["characters"].([]interface{})[0].(map[string]interface{})["position"] = map[string]interface{}{"x": 0, "y": 0}
	data := mustBytes(m)
	ok, details, err := ValidateScenario(data, schema, registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected solid tile rejection")
	}
	if !strings.Contains(strings.Join(details, " "), "solid tile") {
		t.Fatalf("expected solid tile error, got %v", details)
	}
}
