package agent

import (
	"encoding/json"

	"glam/server/llm"
)

var ToolDefs = []llm.ToolDef{
	{
		Name:        "generate_scenario_world",
		Description: "Generate a complete validated GLAM scenario world. Call once you have topic and age_group.",
		Parameters:  json.RawMessage(generateScenarioWorldSchema),
	},
	{
		Name:        "ask_teacher",
		Description: "Ask the teacher for clarification or missing information. Use when topic or age group is missing.",
		Parameters:  json.RawMessage(askTeacherSchema),
	},
}

const generateScenarioWorldSchema = `{"type":"object","required":["topic","age_group"],"additionalProperties":false,"properties":{"topic":{"type":"string","minLength":1,"maxLength":500,"description":"Learning topic, e.g. money management, forest animals"},"age_group":{"type":"string","minLength":1,"maxLength":20,"description":"Age group like 8-10, 10-12"},"template_hint":{"type":"string","enum":["town","forest","desert","school"],"description":"Optional world template preference"}}}`

const askTeacherSchema = `{"type":"object","required":["question"],"additionalProperties":false,"properties":{"question":{"type":"string","minLength":1,"maxLength":500,"description":"Question to ask the teacher"}}}`

func marshalToolResult(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
