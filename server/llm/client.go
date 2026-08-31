package llm

import (
	"bytes"
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
)

func envTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("OPENCODE_TIMEOUT"))
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
	raw := strings.TrimSpace(os.Getenv("OPENCODE_MAX_TOKENS"))
	if raw == "" {
		return defaultMaxTokens
	}
	if v, err := strconv.Atoi(raw); err == nil && v > 0 {
		return v
	}
	return defaultMaxTokens
}

func envPreviewLimit() int {
	raw := strings.TrimSpace(os.Getenv("OPENCODE_ERROR_PREVIEW_LIMIT"))
	if raw == "" {
		return defaultPreviewLimit
	}
	if v, err := strconv.Atoi(raw); err == nil && v > 0 {
		return v
	}
	return defaultPreviewLimit
}

// OpenCodeClient calls the OpenCode Responses endpoint.
type OpenCodeClient struct {
	APIKey   string
	Endpoint string
	Model    string
	HTTPClient *http.Client
}

// NewClient creates a client from env vars with defaults.
func NewClient() *OpenCodeClient {
	key := os.Getenv("OPENCODE_API_KEY")
	endpoint := os.Getenv("OPENCODE_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://opencode.ai/zen/go/v1/responses"
	}
	model := os.Getenv("OPENCODE_MODEL")
	if model == "" {
		model = "muse-spark-1.2-contributor"
	}
	return &OpenCodeClient{
		APIKey:     key,
		Endpoint:   endpoint,
		Model:      model,
		HTTPClient: &http.Client{Timeout: envTimeout()},
	}
}

// Generate calls the Responses API and returns raw JSON string.
func (c *OpenCodeClient) Generate(prompt string, schemaJSON []byte, registryJSON []byte) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("OPENCODE_API_KEY not set")
	}

	systemPrompt := BuildSystemPrompt(schemaJSON, registryJSON)

	var schemaObj interface{}
	if err := json.Unmarshal(schemaJSON, &schemaObj); err != nil {
		return "", fmt.Errorf("invalid schema JSON: %w", err)
	}

	reqBody := map[string]interface{}{
		"model":        c.Model,
		"instructions": systemPrompt,
		"input": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "input_text", "text": prompt},
				},
			},
		},
		"stream": false,
		"store":  false,
		"reasoning": map[string]interface{}{
			"effort": "minimal",
		},
		"max_output_tokens": envMaxTokens(),
		"text": map[string]interface{}{
			"format": map[string]interface{}{
				"type":   "json_schema",
				"name":   "glam_scenario",
				"schema": schemaObj,
				"strict": false,
			},
		},
	}

	// Also include response_format for compatibility
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

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("opencode request failed: %w", err)
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
		return "", fmt.Errorf("opencode API error %d: %s", resp.StatusCode, preview)
	}

	// Try multiple fallback paths to extract JSON text
	text, err := extractText(respBody)
	if err != nil {
		return "", err
	}

	// Trim markdown fences if present
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		// remove fences
		lines := strings.Split(text, "\n")
		// drop first line fence
		if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
			lines = lines[1:]
		}
		// drop last fence if present
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
		text = strings.TrimSpace(strings.Join(lines, "\n"))
		// also handle ```json wrapper where text was ```json\n{...}\n```
		text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "json"))
	}

	// Ensure it is valid JSON by checking
	var js json.RawMessage
	if err := json.Unmarshal([]byte(text), &js); err != nil {
		// Try to locate JSON object inside text
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

	// Path 2: output_text field at top level
	if t, ok := raw["output_text"].(string); ok && t != "" {
		return t, nil
	}

	// Path 3: choices[0].message.content (Chat Completions compat)
	if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
		if cm, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := cm["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok && content != "" {
					return content, nil
				}
				// content as array
				if arr, ok := msg["content"].([]interface{}); ok && len(arr) > 0 {
					if m2, ok := arr[0].(map[string]interface{}); ok {
						if t, ok := m2["text"].(string); ok {
							return t, nil
						}
					}
				}
			}
			if t, ok := cm["text"].(string); ok && t != "" {
				return t, nil
			}
		}
	}

	// Path 4: data or response fields
	if t, ok := raw["data"].(string); ok && t != "" {
		return t, nil
	}
	if t, ok := raw["response"].(string); ok && t != "" {
		return t, nil
	}
	// Path 5: content field
	if t, ok := raw["content"].(string); ok && t != "" {
		return t, nil
	}

	// Fallback: if body itself looks like scenario JSON (has "id" and "world"), return whole body
	if _, hasID := raw["id"]; hasID {
		if _, hasWorld := raw["world"]; hasWorld {
			return string(body), nil
		}
	}

	return "", fmt.Errorf("unable to extract text from response; body preview: %s", truncate(string(body), envPreviewLimit()))
}
