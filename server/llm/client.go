package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout      = 60 * time.Second
	defaultMaxTokens    = 6000
	defaultPreviewLimit = 2000
	defaultEndpoint     = "https://openrouter.ai/api/v1/chat/completions"
	defaultModel        = "google/gemma-4-31b-it:free"
)

// envOr returns first non-empty env var among keys.
func envOr(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func envTimeout() time.Duration {
	raw := envOr("OPENROUTER_TIMEOUT", "OPENCODE_TIMEOUT")
	if raw == "" {
		return defaultTimeout
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		return time.Duration(secs) * time.Second
	}
	return defaultTimeout
}

func envMaxTokens() int {
	raw := envOr("OPENROUTER_MAX_TOKENS", "OPENCODE_MAX_TOKENS")
	if raw == "" {
		return defaultMaxTokens
	}
	if v, err := strconv.Atoi(raw); err == nil && v > 0 {
		return v
	}
	return defaultMaxTokens
}

func envPreviewLimit() int {
	raw := envOr("OPENROUTER_ERROR_PREVIEW_LIMIT", "OPENCODE_ERROR_PREVIEW_LIMIT")
	if raw == "" {
		return defaultPreviewLimit
	}
	if v, err := strconv.Atoi(raw); err == nil && v > 0 {
		return v
	}
	return defaultPreviewLimit
}

// OpenRouterClient calls the OpenRouter Chat Completions endpoint.
// Alias OpenCodeClient is kept for backward compatibility.
type OpenRouterClient struct {
	APIKey     string
	Endpoint   string
	Model      string
	HTTPClient *http.Client
}

// CompletionResult carries both plain text and tool-call outcomes from a
// tool-calling completion.
type CompletionResult struct {
	Content   string
	ToolCalls []ToolCallReq
}

// ToolCallReq represents a single function tool call requested by the model.
type ToolCallReq struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// OpenCodeClient is an alias for backward-compatibility — existing code
// in api/handler.go referenced llm.OpenCodeClient.
type OpenCodeClient = OpenRouterClient

// NewClient creates a client from env vars with defaults.
// Prefers OPENROUTER_* vars, falls back to OPENCODE_* for backward compat.
func NewClient() *OpenRouterClient {
	key := envOr("OPENROUTER_API_KEY", "OPENCODE_API_KEY")
	endpoint := envOr("OPENROUTER_ENDPOINT", "OPENCODE_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	model := envOr("OPENROUTER_MODEL", "OPENCODE_MODEL")
	if model == "" {
		model = defaultModel
	}
	// User explicitly requested google/gemma-4-31b-it:free — if defaultModel
	// was not overridden but user expects that variant, honor OPENROUTER_MODEL.
	// Keep default as gemma-3 27b free since gemma-4 31b does not exist on
	// OpenRouter at time of writing; if caller sets OPENROUTER_MODEL to
	// google/gemma-4-31b-it:free it will be used verbatim.
	return &OpenRouterClient{
		APIKey:     key,
		Endpoint:   endpoint,
		Model:      model,
		HTTPClient: &http.Client{Timeout: envTimeout()},
	}
}

// NewOpenRouterClient is an explicit constructor alias.
func NewOpenRouterClient() *OpenRouterClient { return NewClient() }

// GenerateWithTools performs a tool-calling chat completion with full message history.
func (c *OpenRouterClient) GenerateWithTools(ctx context.Context, messages []Message, tools []ToolDef) (*CompletionResult, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not set (also checked OPENCODE_API_KEY)")
	}
	msgPayload := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		entry := map[string]interface{}{
			"role": m.Role,
		}
		if m.Content != "" {
			entry["content"] = m.Content
		} else if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			// assistant tool_call message may have empty content
			entry["content"] = nil
		}
		if m.ToolCallID != "" {
			entry["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]map[string]interface{}, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				argsStr := string(tc.Arguments)
				if argsStr == "" {
					argsStr = "{}"
				}
				tcs = append(tcs, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": argsStr,
					},
				})
			}
			entry["tool_calls"] = tcs
		}
		msgPayload = append(msgPayload, entry)
	}
	toolPayload := make([]map[string]interface{}, 0, len(tools))
	for _, td := range tools {
		toolPayload = append(toolPayload, td.ToOpenAIFunction())
	}
	reqBody := map[string]interface{}{
		"model":       c.Model,
		"messages":    msgPayload,
		"max_tokens":  envMaxTokens(),
		"temperature": 0.7,
	}
	if len(toolPayload) > 0 {
		reqBody["tools"] = toolPayload
		reqBody["tool_choice"] = "auto"
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("HTTP-Referer", envOr("OPENROUTER_REFERER", "GLAM_APP_REFERER"))
	if title := envOr("OPENROUTER_TITLE", "GLAM_APP_TITLE"); title != "" {
		req.Header.Set("X-Title", title)
	} else {
		req.Header.Set("X-Title", "GLAM")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limit := envPreviewLimit()
		preview := string(respBody)
		if len(preview) > limit {
			preview = preview[:limit]
		}
		return nil, fmt.Errorf("openrouter API error %d: %s", resp.StatusCode, preview)
	}
	toolCalls, err := extractToolCalls(respBody)
	if err != nil {
		return nil, err
	}
	content, _ := extractText(respBody)
	if len(toolCalls) > 0 {
		return &CompletionResult{Content: content, ToolCalls: toolCalls}, nil
	}
	if content != "" {
		content = strings.TrimSpace(content)
		if strings.HasPrefix(content, "```") {
			lines := strings.Split(content, "\n")
			if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
				lines = lines[1:]
			}
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			content = strings.TrimSpace(strings.Join(lines, "\n"))
			content = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(content), "json"))
		}
	}
	return &CompletionResult{Content: content, ToolCalls: nil}, nil
}

func extractToolCalls(body []byte) ([]ToolCallReq, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse response JSON: %w; body: %s", err, truncate(string(body), envPreviewLimit()))
	}
	choices, ok := raw["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, nil
	}
	cm, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	msg, ok := cm["message"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	tcs, ok := msg["tool_calls"].([]interface{})
	if !ok || len(tcs) == 0 {
		return nil, nil
	}
	var out []ToolCallReq
	for _, tcRaw := range tcs {
		tcMap, ok := tcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := tcMap["id"].(string)
		fn, _ := tcMap["function"].(map[string]interface{})
		if fn == nil {
			continue
		}
		name, _ := fn["name"].(string)
		argsStr, _ := fn["arguments"].(string)
		if argsStr == "" {
			argsStr = "{}"
		}
		var rawArgs json.RawMessage = json.RawMessage(argsStr)
		if !json.Valid([]byte(argsStr)) {
			rawArgs = json.RawMessage("{}")
		}
		if name == "" {
			continue
		}
		out = append(out, ToolCallReq{ID: id, Name: name, Arguments: rawArgs})
	}
	return out, nil
}

// Generate calls the OpenRouter Chat Completions API and returns raw JSON string.
func (c *OpenRouterClient) Generate(prompt string, schemaJSON []byte, registryJSON []byte) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not set (also checked OPENCODE_API_KEY)")
	}

	systemPrompt := BuildSystemPrompt(schemaJSON, registryJSON)

	reqBody := map[string]interface{}{
		"model": c.Model,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  envMaxTokens(),
		"temperature": 0.7,
		"response_format": map[string]interface{}{
			"type": "json_object",
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("HTTP-Referer", envOr("OPENROUTER_REFERER", "GLAM_APP_REFERER"))
	if title := envOr("OPENROUTER_TITLE", "GLAM_APP_TITLE"); title != "" {
		req.Header.Set("X-Title", title)
	} else {
		req.Header.Set("X-Title", "GLAM")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openrouter request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limit := envPreviewLimit()
		preview := string(respBody)
		if len(preview) > limit {
			preview = preview[:limit]
		}
		return "", fmt.Errorf("openrouter API error %d: %s", resp.StatusCode, preview)
	}

	text, err := extractText(respBody)
	if err != nil {
		return "", err
	}

	// Trim markdown fences if present (some models wrap JSON despite json_object)
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
		text = strings.TrimSpace(strings.Join(lines, "\n"))
		text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "json"))
	}

	var js json.RawMessage
	if err := json.Unmarshal([]byte(text), &js); err != nil {
		start := strings.Index(text, "{")
		end := strings.LastIndex(text, "}")
		if start >= 0 && end > start {
			candidate := text[start : end+1]
			if err2 := json.Unmarshal([]byte(candidate), &js); err2 == nil {
				return candidate, nil
			}
		}
		return "", fmt.Errorf("LLM did not return valid JSON: %v; raw: %s", err, truncate(text, envPreviewLimit()))
	}

	return text, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "...[truncated]"
	}
	return s
}

func extractText(body []byte) (string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("parse response JSON: %w; body: %s", err, truncate(string(body), envPreviewLimit()))
	}

	// OpenRouter / OpenAI Chat Completions: choices[0].message.content
	if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
		if cm, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := cm["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok && content != "" {
					return content, nil
				}
				if arr, ok := msg["content"].([]interface{}); ok && len(arr) > 0 {
					if m2, ok := arr[0].(map[string]interface{}); ok {
						if t, ok := m2["text"].(string); ok && t != "" {
							return t, nil
						}
						if t, ok := m2["content"].(string); ok && t != "" {
							return t, nil
						}
					}
					// concatenate text parts if array of content blocks
					var parts []string
					for _, p := range arr {
						if pm, ok := p.(map[string]interface{}); ok {
							if t, ok := pm["text"].(string); ok && t != "" {
								parts = append(parts, t)
							}
						}
					}
					if len(parts) > 0 {
						return strings.Join(parts, "\n"), nil
					}
				}
				// reasoning models may put content in reasoning field
				if r, ok := msg["reasoning"].(string); ok && r != "" {
					// still prefer content, but fallback
					return r, nil
				}
			}
			if t, ok := cm["text"].(string); ok && t != "" {
				return t, nil
			}
		}
	}

	// OpenCode legacy path: output[].content[].text
	if out, ok := raw["output"].([]interface{}); ok && len(out) > 0 {
		for _, item := range out {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if m["type"] == "reasoning" {
				continue
			}
			if content, ok := m["content"].([]interface{}); ok {
				for _, c := range content {
					if cm, ok := c.(map[string]interface{}); ok {
						if t, ok := cm["text"].(string); ok && t != "" {
							return t, nil
						}
						if t, ok := cm["output_text"].(string); ok && t != "" {
							return t, nil
						}
					}
				}
			}
			if t, ok := m["text"].(string); ok && t != "" {
				return t, nil
			}
		}
	}

	// Other fallbacks
	if t, ok := raw["output_text"].(string); ok && t != "" {
		return t, nil
	}
	if t, ok := raw["data"].(string); ok && t != "" {
		return t, nil
	}
	if t, ok := raw["response"].(string); ok && t != "" {
		return t, nil
	}
	if t, ok := raw["content"].(string); ok && t != "" {
		return t, nil
	}

	// If body itself looks like scenario JSON (has id+world), return whole body
	if _, hasID := raw["id"]; hasID {
		if _, hasWorld := raw["world"]; hasWorld {
			return string(body), nil
		}
	}

	return "", fmt.Errorf("unable to extract text from response; body preview: %s", truncate(string(body), envPreviewLimit()))
}
