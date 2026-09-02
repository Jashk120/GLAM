package scenario

import (
	"strings"
	"testing"
)

func validArena() map[string]interface{} {
	return map[string]interface{}{
		"version": "1",
		"id":      "adding-apples",
		"title":   "Adding Apples",
		"theme":   "meadow",
		"cast": map[string]interface{}{
			"student": map[string]interface{}{"variant": "girl", "name": "Ava"},
			"mascot":  map[string]interface{}{"id": "nova-fox", "name": "Nova", "side": "right"},
		},
		"flow": map[string]interface{}{
			"start": "welcome",
			"nodes": map[string]interface{}{
				"welcome": map[string]interface{}{
					"id": "welcome", "type": "dialogue", "speaker": "mascot",
					"text": "Let us count apples.", "next": "show-apples",
				},
				"show-apples": map[string]interface{}{
					"id": "show-apples", "type": "teaching",
					"stage": map[string]interface{}{
						"visual": map[string]interface{}{"type": "countingObjects", "object": "apple", "count": 3},
						"motion": []interface{}{map[string]interface{}{"type": "add", "count": 2, "target": "stage"}},
					},
					"next": "answer",
				},
				"answer": map[string]interface{}{
					"id": "answer", "type": "multipleChoice", "prompt": "How many apples?",
					"options": []interface{}{
						map[string]interface{}{"id": "four", "text": "4"},
						map[string]interface{}{"id": "five", "text": "5"},
					},
					"correctOptionIds": []interface{}{"five"},
					"feedback": map[string]interface{}{
						"correct":   map[string]interface{}{"mascotText": "Correct!", "next": "complete"},
						"incorrect": map[string]interface{}{"mascotText": "Try again.", "retryPolicy": "afterHint"},
					},
				},
				"complete": map[string]interface{}{"id": "complete", "type": "complete", "title": "Great work", "summary": "You added apples."},
			},
		},
	}
}

func TestValidateScenario_Arena(t *testing.T) {
	schema, registry := testPaths()
	valid := validMap()
	valid["arena"] = validArena()
	ok, details, err := ValidateScenario(mustBytes(valid), schema, registry)
	if err != nil {
		t.Fatalf("unexpected validator error: %v", err)
	}
	if !ok {
		t.Fatalf("expected valid arena, got %v", details)
	}

	invalid := validMap()
	arena := validArena()
	nodes := arena["flow"].(map[string]interface{})["nodes"].(map[string]interface{})
	nodes["welcome"].(map[string]interface{})["next"] = "missing"
	invalid["arena"] = arena
	ok, details, err = ValidateScenario(mustBytes(invalid), schema, registry)
	if err != nil {
		t.Fatalf("unexpected validator error: %v", err)
	}
	if ok || !strings.Contains(strings.Join(details, " "), "missing") {
		t.Fatalf("expected missing node rejection, got ok=%v details=%v", ok, details)
	}
}
