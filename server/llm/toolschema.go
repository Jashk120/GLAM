package llm

import "encoding/json"

// ToolDef describes a function tool in OpenAI/OpenRouter shape.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// ToOpenAIFunction returns the OpenAI function-calling shape:
// {"type":"function","function":{"name":..., "description":..., "parameters":...}}
func (t ToolDef) ToOpenAIFunction() map[string]interface{} {
	fn := map[string]interface{}{
		"name":        t.Name,
		"description": t.Description,
	}
	if len(t.Parameters) > 0 {
		var params interface{}
		if err := json.Unmarshal(t.Parameters, &params); err == nil {
			fn["parameters"] = params
		} else {
			fn["parameters"] = json.RawMessage(t.Parameters)
		}
	} else {
		fn["parameters"] = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}
	return map[string]interface{}{
		"type":     "function",
		"function": fn,
	}
}
