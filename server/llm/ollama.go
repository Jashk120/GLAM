package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOllamaEndpoint = "http://127.0.0.1:11434/api/chat"
	defaultOllamaModel    = "qwen2.5-coder:14b"
	defaultOllamaTimeout  = 180 * time.Second
)

// OllamaClient calls a locally running Ollama chat endpoint. Ollama does not
// need an API key when it is used on the local machine.
type OllamaClient struct {
	Endpoint   string
	Model      string
	HTTPClient *http.Client
}

func NewOllamaClient() *OllamaClient {
	endpoint := envOr("OLLAMA_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultOllamaEndpoint
	}
	model := envOr("OLLAMA_MODEL")
	if model == "" {
		model = defaultOllamaModel
	}
	return &OllamaClient{Endpoint: endpoint, Model: model, HTTPClient: &http.Client{Timeout: envOllamaTimeout()}}
}

func envOllamaTimeout() time.Duration {
	raw := envOr("OLLAMA_TIMEOUT")
	if raw == "" {
		return defaultOllamaTimeout
	}
	if timeout, err := time.ParseDuration(raw); err == nil {
		return timeout
	}
	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return defaultOllamaTimeout
}

func (c *OllamaClient) IsConfigured() bool {
	return strings.TrimSpace(c.Endpoint) != "" && strings.TrimSpace(c.Model) != ""
}

func (c *OllamaClient) Generate(prompt string, schemaJSON []byte, registryJSON []byte) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("Ollama is not configured: set OLLAMA_MODEL")
	}
	return c.generate(BuildSystemPrompt(schemaJSON, registryJSON), prompt, 0.7)
}

// Repair asks the local model for one schema-correct replacement after an
// initial response fails validation. It is deliberately not a loop.
func (c *OllamaClient) Repair(invalidJSON string, details []string, schemaJSON []byte, registryJSON []byte) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("Ollama is not configured: set OLLAMA_MODEL")
	}
	systemPrompt := "You repair rejected GLAM scenario JSON. Return only a complete replacement JSON object. " + BuildSystemPrompt(schemaJSON, registryJSON)
	userPrompt := fmt.Sprintf("The previous JSON was rejected. Fix every listed issue; do not explain.\nValidation errors:\n- %s\nRejected JSON:\n%s", strings.Join(details, "\n- "), invalidJSON)
	return c.generate(systemPrompt, userPrompt, 0.1)
}

func (c *OllamaClient) generate(systemPrompt, userPrompt string, temperature float64) (string, error) {
	reqBody := map[string]interface{}{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"stream":  false,
		"format":  "json",
		"options": map[string]interface{}{"temperature": temperature, "num_predict": envMaxTokens()},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal Ollama request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create Ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: envOllamaTimeout()}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read Ollama response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("Ollama API error %d: %s", resp.StatusCode, truncate(string(responseBody), envPreviewLimit()))
	}
	var response struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("parse Ollama response: %w", err)
	}
	if strings.TrimSpace(response.Message.Content) == "" {
		return "", fmt.Errorf("Ollama returned an empty response")
	}
	return normalizeScenarioJSON(response.Message.Content)
}
