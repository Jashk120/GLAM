package llm

import (
	"strings"
	"testing"
)

func TestBuildSystemPromptKeepsArenaNestedInScenarioRoot(t *testing.T) {
	prompt := BuildSystemPrompt([]byte(`{"type":"object"}`), []byte(`[]`))
	for _, required := range []string{
		"ROOT OBJECT — REQUIRED FOR EVERY GAME",
		`"characters":[]`,
		"arena is an OPTIONAL field inside this root object",
		"Never put characters, objects, background, or activities inside arena",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("prompt missing %q", required)
		}
	}
}
