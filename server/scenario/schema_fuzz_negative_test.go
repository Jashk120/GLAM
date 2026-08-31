package scenario

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// schema_fuzz_negative_test.go — schema-level fuzzing for GLAM Scenario JSON.
// Interaction fuzzing + strict bounds (additionalProperties:false, propertyNames blocklist).
// Uses jsonschema Draft2020 directly; no network, deterministic.

func compileTestSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	sch, err := c.Compile("../../schema/scenario.schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func validateDoc(sch *jsonschema.Schema, doc interface{}) error {
	b, _ := json.Marshal(doc)
	var v interface{}
	_ = json.Unmarshal(b, &v)
	return sch.Validate(v)
}

func mustValidate(t *testing.T, sch *jsonschema.Schema, doc interface{}, shouldPass bool, substr string) {
	t.Helper()
	err := validateDoc(sch, doc)
	if shouldPass && err != nil {
		t.Fatalf("expected pass got %v doc=%v", err, doc)
	}
	if !shouldPass && err == nil {
		t.Fatalf("expected fail %q got pass doc=%v", substr, doc)
	}
	if !shouldPass && substr != "" {
		low := strings.ToLower(substr)
		// flatten ValidationError causes for oneOf (dialogue/mcq/etc. obscures specific keyword)
		var joined string
		if ve, ok := err.(*jsonschema.ValidationError); ok {
			var msgs []string
			var walk func(*jsonschema.ValidationError)
			walk = func(v *jsonschema.ValidationError) {
				if len(v.Causes) == 0 {
					msgs = append(msgs, v.Error())
					return
				}
				for _, c := range v.Causes {
					walk(c)
				}
			}
			walk(ve)
			joined = strings.ToLower(strings.Join(msgs, " | ") + " | " + err.Error())
		} else {
			joined = strings.ToLower(err.Error())
		}
		if !strings.Contains(joined, low) {
			t.Fatalf("expected %q in %q doc=%v", substr, err.Error(), doc)
		}
	}
}

func baseScenario() map[string]interface{} {
	return map[string]interface{}{
		"id": "test-scenario", "title": "Test",
		"world":      map[string]interface{}{"template": "town", "spawn": map[string]interface{}{"x": 1, "y": 1}, "size": map[string]interface{}{"cols": 15, "rows": 12}},
		"characters": []interface{}{}, "buildings": []interface{}{}, "objects": []interface{}{}, "missions": []interface{}{},
	}
}

func charWithInteraction(m map[string]interface{}) []interface{} {
	return []interface{}{map[string]interface{}{"id": "c1", "name": "Alice", "position": map[string]interface{}{"x": 1, "y": 1}, "interaction": m}}
}

func TestSchemaPositiveControl(t *testing.T) {
	sch := compileTestSchema(t)
	mustValidate(t, sch, baseScenario(), true, "")
	data, err := os.ReadFile("../../scenarios/example.json")
	if err != nil {
		t.Fatalf("read example.json: %v", err)
	}
	var doc interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse example.json: %v", err)
	}
	if err := sch.Validate(doc); err != nil {
		t.Fatalf("example.json should pass schema: %v", err)
	}
	for _, price := range []int{0, 1000000} {
		m := baseScenario()
		m["characters"] = charWithInteraction(map[string]interface{}{"type": "shop", "items": []interface{}{map[string]interface{}{"name": "Item", "price": price}}})
		mustValidate(t, sch, m, true, "")
	}
}

func TestSchemaFuzzNegative(t *testing.T) {
	sch := compileTestSchema(t)
	long1001 := strings.Repeat("a", 1001)
	long201 := strings.Repeat("b", 201)
	type tc struct {
		name       string
		mutate     func(map[string]interface{})
		shouldPass bool
		substr     string
	}
	tests := []tc{
		{"dialogue missing text", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue"})
		}, false, "required"},
		{"mcq missing question", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "mcq", "options": []interface{}{map[string]interface{}{"text": "A", "correct": true}, map[string]interface{}{"text": "B", "correct": false}}})
		}, false, "required"},
		{"mcq missing options", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "mcq", "question": "Q?"})
		}, false, "required"},
		{"math missing answer", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "math", "question": "2+2?"})
		}, false, "required"},
		{"shop missing items", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "shop"})
		}, false, "required"},
		{"information missing content", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "information"})
		}, false, "required"},
		{"mcq options len 1", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "mcq", "question": "Q?", "options": []interface{}{map[string]interface{}{"text": "A", "correct": true}}})
		}, false, "minItems"},
		{"mcq options len 6", func(m map[string]interface{}) {
			o := []interface{}{}
			for i := 0; i < 6; i++ {
				o = append(o, map[string]interface{}{"text": "O", "correct": false})
			}
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "mcq", "question": "Q?", "options": o})
		}, false, "maxItems"},
		{"mcq option missing text", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "mcq", "question": "Q?", "options": []interface{}{map[string]interface{}{"correct": true}, map[string]interface{}{"text": "B", "correct": false}}})
		}, false, "required"},
		{"mcq option missing correct", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "mcq", "question": "Q?", "options": []interface{}{map[string]interface{}{"text": "A"}, map[string]interface{}{"text": "B", "correct": false}}})
		}, false, "required"},
		{"shop items len 0", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "shop", "items": []interface{}{}})
		}, false, "minItems"},
		{"shop items len 11", func(m map[string]interface{}) {
			it := []interface{}{}
			for i := 0; i < 11; i++ {
				it = append(it, map[string]interface{}{"name": "N", "price": 1})
			}
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "shop", "items": it})
		}, false, "maxItems"},
		{"shop price -1", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "shop", "items": []interface{}{map[string]interface{}{"name": "Bad", "price": -1}}})
		}, false, "minimum"},
		{"shop price 1e7 over max", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "shop", "items": []interface{}{map[string]interface{}{"name": "Exp", "price": 10000000}}})
		}, false, "maximum"},
		{"shop name empty", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "shop", "items": []interface{}{map[string]interface{}{"name": "", "price": 5}}})
		}, false, "minLength"},
		{"dialogue text empty", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": ""})
		}, false, "minLength"},
		{"dialogue text 1001 chars", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": long1001})
		}, false, "maxLength"},
		{"math tolerance -1", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "math", "question": "Q?", "answer": 42, "tolerance": -1})
		}, false, "minimum"},
		{"cooldown -1", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": "hi", "cooldown": -1})
		}, false, "minimum"},
		{"auto not boolean", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": "hi", "auto": "yes"})
		}, false, "type"},
		{"onCorrect stat uppercase", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": "hi", "onCorrect": map[string]interface{}{"stat": "Coins"}})
		}, false, "pattern"},
		{"onCorrect stat hyphen", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": "hi", "onCorrect": map[string]interface{}{"stat": "my-stat"}})
		}, false, "pattern"},
		{"onCorrect stat empty", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": "hi", "onCorrect": map[string]interface{}{"stat": ""}})
		}, false, "minLength"},
		{"delta not int", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": "hi", "onCorrect": map[string]interface{}{"delta": 1.5}})
		}, false, "type"},
		{"toast empty", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": "hi", "onCorrect": map[string]interface{}{"toast": ""}})
		}, false, "minLength"},
		{"toast 201 chars", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": "hi", "onCorrect": map[string]interface{}{"toast": long201}})
		}, false, "maxLength"},
		{"additionalProperties foo at interaction", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "dialogue", "text": "hi", "foo": "bar"})
		}, false, "additionalProperties"},
		{"additionalProperties reward at mission", func(m map[string]interface{}) {
			m["missions"] = []interface{}{map[string]interface{}{"id": "m1", "title": "T", "description": "D", "reward": 10}}
		}, false, "additionalProperties"},
		{"propertyNames code at building", func(m map[string]interface{}) {
			m["buildings"] = []interface{}{map[string]interface{}{"id": "b1", "typeAssetId": "shop_small", "position": map[string]interface{}{"x": 1, "y": 1}, "code": "evil"}}
		}, false, "propertyNames"},
		{"propertyNames Script at character cap", func(m map[string]interface{}) {
			m["characters"] = []interface{}{map[string]interface{}{"id": "c1", "name": "A", "position": map[string]interface{}{"x": 1, "y": 1}, "Script": "x"}}
		}, false, "additionalProperties"},
		{"propertyNames bundle at building", func(m map[string]interface{}) {
			m["buildings"] = []interface{}{map[string]interface{}{"id": "b1", "typeAssetId": "shop_small", "position": map[string]interface{}{"x": 1, "y": 1}, "bundle": "evil"}}
		}, false, "propertyNames"},
		{"version v1 invalid", func(m map[string]interface{}) { v := "v1"; m["version"] = v }, false, "pattern"},
		{"version 1 invalid", func(m map[string]interface{}) { v := "1"; m["version"] = v }, false, "pattern"},
		{"version empty invalid", func(m map[string]interface{}) { v := ""; m["version"] = v }, false, "minLength"},
		{"initialStats coins -1", func(m map[string]interface{}) { m["initialStats"] = map[string]interface{}{"coins": -1} }, false, "minimum"},
		{"initialStats coins 1e7 over max", func(m map[string]interface{}) { m["initialStats"] = map[string]interface{}{"coins": 10000000} }, false, "maximum"},
		{"initialStats lives 100 over max", func(m map[string]interface{}) { m["initialStats"] = map[string]interface{}{"lives": 100} }, false, "maximum"},
		{"initialStats score -1", func(m map[string]interface{}) { m["initialStats"] = map[string]interface{}{"score": -1} }, false, "minimum"},
		{"regions 21 over max", func(m map[string]interface{}) {
			r := []interface{}{}
			for i := 0; i < 21; i++ {
				r = append(r, map[string]interface{}{"id": "r"})
			}
			m["world"] = map[string]interface{}{"template": "town", "spawn": map[string]interface{}{"x": 1, "y": 1}, "size": map[string]interface{}{"cols": 15, "rows": 12}, "regions": r}
		}, false, "maxItems"},
		{"region width 0", func(m map[string]interface{}) {
			m["world"] = map[string]interface{}{"template": "town", "spawn": map[string]interface{}{"x": 1, "y": 1}, "size": map[string]interface{}{"cols": 15, "rows": 12}, "regions": []interface{}{map[string]interface{}{"id": "r1", "width": 0}}}
		}, false, "minimum"},
		{"information image empty", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "information", "content": "hi", "image": ""})
		}, false, "minLength"},
		{"currency empty", func(m map[string]interface{}) {
			m["characters"] = charWithInteraction(map[string]interface{}{"type": "shop", "currency": "", "items": []interface{}{map[string]interface{}{"name": "N", "price": 1}}})
		}, false, "minLength"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := baseScenario()
			tc.mutate(doc)
			mustValidate(t, sch, doc, tc.shouldPass, tc.substr)
		})
	}
}
