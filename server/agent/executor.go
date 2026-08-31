package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"glam/server/llm"
	"glam/server/pipeline"
)

func ExecuteTool(ctx context.Context, deps pipeline.Deps, client *llm.OpenRouterClient, call llm.ToolCallReq) (string, bool, error) {
	switch call.Name {
	case "generate_scenario_world":
		var args struct {
			Topic        string  `json:"topic"`
			AgeGroup     string  `json:"age_group"`
			TemplateHint *string `json:"template_hint"`
		}
		if err := json.Unmarshal(call.Arguments, &args); err != nil {
			return marshalToolResult(map[string]interface{}{"error": fmt.Sprintf("invalid arguments: %v", err)}), false, nil
		}
		if args.Topic == "" || args.AgeGroup == "" {
			return marshalToolResult(map[string]interface{}{"error": "topic and age_group are required"}), false, nil
		}
		hint := ""
		if args.TemplateHint != nil {
			hint = *args.TemplateHint
		}
		scenario, warnings, err := pipeline.GenerateScenarioWorld(ctx, deps, args.Topic, args.AgeGroup, hint, client)
		if err != nil {
			return marshalToolResult(map[string]interface{}{"error": err.Error()}), false, nil
		}
		result := map[string]interface{}{
			"scenario": scenario,
		}
		if len(warnings) > 0 {
			result["warnings"] = warnings
		}
		return marshalToolResult(result), false, nil

	case "ask_teacher":
		var args struct {
			Question string `json:"question"`
		}
		if err := json.Unmarshal(call.Arguments, &args); err != nil {
			return marshalToolResult(map[string]interface{}{"question": "Could you provide more details?"}), true, nil
		}
		q := args.Question
		if q == "" {
			q = "Could you clarify?"
		}
		return marshalToolResult(map[string]interface{}{"question": q}), true, nil

	default:
		return marshalToolResult(map[string]interface{}{"error": fmt.Sprintf("unknown tool %q", call.Name)}), false, nil
	}
}
