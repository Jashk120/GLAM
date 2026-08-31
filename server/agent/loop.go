package agent

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"glam/server/llm"
	"glam/server/pipeline"
)

func maxToolIterations() int {
	raw := os.Getenv("AGENT_MAX_TOOL_ITERATIONS")
	if raw == "" {
		return 4
	}
	if v, err := strconv.Atoi(raw); err == nil && v > 0 {
		return v
	}
	return 4
}

func RunTurn(ctx context.Context, client *llm.OpenRouterClient, history []llm.Message, deps pipeline.Deps) ([]llm.Message, error) {
	maxIters := maxToolIterations()
	updated := make([]llm.Message, len(history))
	copy(updated, history)

	for i := 0; i < maxIters; i++ {
		result, err := client.GenerateWithTools(ctx, updated, ToolDefs)
		if err != nil {
			return updated, fmt.Errorf("llm error: %w", err)
		}
		if len(result.ToolCalls) == 0 {
			updated = append(updated, llm.NewAssistantMessage(result.Content))
			return updated, nil
		}
		updated = append(updated, llm.Message{Role: "assistant", ToolCalls: result.ToolCalls})
		for _, call := range result.ToolCalls {
			out, stop, execErr := ExecuteTool(ctx, deps, client, call)
			if execErr != nil {
				out = marshalToolResult(map[string]interface{}{"error": execErr.Error()})
			}
			updated = append(updated, llm.NewToolResultMessage(call.ID, out))
			if stop {
				return updated, nil
			}
		}
	}
	return updated, fmt.Errorf("max tool iterations (%d) exceeded", maxIters)
}
