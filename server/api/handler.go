package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"glam/server/llm"
	"glam/server/scenario"
)

type Handler struct {
	SchemaPath   string
	RegistryPath string
	SchemaJSON   []byte
	RegistryJSON []byte
	LLM          *llm.OpenCodeClient
}

func NewHandler(schemaPath, registryPath string) (*Handler, error) {
	schemaJSON, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}
	_, registryJSON, err := scenario.LoadRegistry(registryPath)
	if err != nil {
		return nil, fmt.Errorf("load registry: %w", err)
	}
	// Also read raw registry JSON for prompt
	rawReg, _ := os.ReadFile(registryPath)
	if len(rawReg) > 0 {
		registryJSON = rawReg
	}

	return &Handler{
		SchemaPath:   schemaPath,
		RegistryPath: registryPath,
		SchemaJSON:   schemaJSON,
		RegistryJSON: registryJSON,
		LLM:          llm.NewClient(),
	}, nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string, details []string) {
	resp := map[string]interface{}{
		"error": msg,
	}
	if len(details) > 0 {
		resp["details"] = details
	}
	writeJSON(w, status, resp)
}

// HandleGenerate POST /api/scenario/generate { prompt: string }
func (h *Handler) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}

	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", []string{err.Error()})
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if len(prompt) < 1 || len(prompt) > 2000 {
		writeError(w, http.StatusBadRequest, "prompt must be 1-2000 characters", nil)
		return
	}

	if h.LLM.APIKey == "" {
		log.Println("OPENROUTER_API_KEY not set (also checked OPENCODE_API_KEY)")
		writeError(w, http.StatusInternalServerError, "server not configured: OPENROUTER_API_KEY missing", nil)
		return
	}

	rawJSON, err := h.LLM.Generate(prompt, h.SchemaJSON, h.RegistryJSON)
	if err != nil {
		// Do not leak API key
		log.Printf("LLM generate error: %v", err)
		// Distinguish provider errors as 502
		msg := err.Error()
		if strings.Contains(msg, "OPENROUTER_API_KEY") || strings.Contains(msg, "OPENCODE_API_KEY") {
			writeError(w, http.StatusInternalServerError, "server configuration error", nil)
			return
		}
		writeError(w, http.StatusBadGateway, "LLM request failed", []string{truncateErr(msg)})
		return
	}

	// Validation pipeline with auto-repair fallback for plot mismatches
	rawBytes := []byte(rawJSON)
	ok, details, err := scenario.ValidateScenario(rawBytes, h.SchemaPath, h.RegistryPath)
	if err != nil {
		log.Printf("validation internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "validation internal error", nil)
		return
	}
	if !ok {
		// Best-effort auto-fix for template-locked plot IDs (e.g. clearing_2 in town)
		if hasPlotError(details) {
			if fixed, didFix, ferr := scenario.NormalizePlotRefs(rawBytes); ferr == nil && didFix {
				if ok2, details2, err2 := scenario.ValidateScenario(fixed, h.SchemaPath, h.RegistryPath); err2 == nil && ok2 {
					log.Printf("auto-fixed plot refs (was %v) — retry validation passed", details)
					rawBytes = fixed
					rawJSON = string(fixed)
					ok = true
					details = details2
				} else if err2 == nil && !ok2 {
					// Still failing but maybe fewer errors — log and keep original details for response
					log.Printf("auto-fix attempted but still invalid: %v", details2)
				}
			}
		}
		if !ok {
			writeError(w, http.StatusBadRequest, "generated scenario failed validation", details)
			return
		}
	}

	// Parse to ensure object, then return
	var scenarioObj map[string]interface{}
	if err := json.Unmarshal(rawBytes, &scenarioObj); err != nil {
		writeError(w, http.StatusBadRequest, "generated scenario is not valid JSON", []string{err.Error()})
		return
	}

	// Optionally save to scenarios/generated_*.json
	_ = saveGenerated(scenarioObj)

	resp := map[string]interface{}{
		"scenario": scenarioObj,
	}
	// include raw for debugging if needed via query param? Spec says raw?: string optional
	// we omit unless requested
	if r.URL.Query().Get("raw") == "1" {
		resp["raw"] = rawJSON
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleValidate GET/POST /api/scenario/validate helper
func (h *Handler) HandleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var data []byte
	if r.Method == http.MethodPost {
		// Accept either raw scenario JSON or { scenario: {...} }
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "read body failed", []string{err.Error()})
			return
		}
		// Try to unwrap { scenario: ... }
		var wrapper map[string]json.RawMessage
		if err := json.Unmarshal(body, &wrapper); err == nil {
			if inner, ok := wrapper["scenario"]; ok {
				data = inner
			} else {
				data = body
			}
		} else {
			data = body
		}
	} else if r.Method == http.MethodGet {
		// Validate example file
		examplePath := filepath.Join(filepath.Dir(h.SchemaPath), "..", "scenarios", "example.json")
		b, err := os.ReadFile(examplePath)
		if err != nil {
			// fallback to current dir
			b, err = os.ReadFile("scenarios/example.json")
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to read example scenario", nil)
				return
			}
		}
		data = b
	} else {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}

	ok, details, err := scenario.ValidateScenario(data, h.SchemaPath, h.RegistryPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "validation error", []string{err.Error()})
		return
	}
	if !ok {
		writeError(w, http.StatusBadRequest, "validation failed", details)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}

// HandleAssets GET /api/assets
func (h *Handler) HandleAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	var assets interface{}
	if err := json.Unmarshal(h.RegistryJSON, &assets); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse registry", nil)
		return
	}
	writeJSON(w, http.StatusOK, assets)
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	var data json.RawMessage
	// limit size 2MB
	r.Body = http.MaxBytesReader(nil, r.Body, 2<<20)
	buf := new(strings.Builder)
	// Use io.ReadAll via json decode workaround: read directly
	// Simpler: decode to raw
	// We need raw bytes, so read
	body := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			body = append(body, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	_ = data
	_ = buf
	if len(body) == 0 {
		return nil, fmt.Errorf("empty body")
	}
	return body, nil
}

func saveGenerated(obj map[string]interface{}) error {
	id, _ := obj["id"].(string)
	if id == "" {
		id = fmt.Sprintf("generated_%d", time.Now().Unix())
	}
	// Ensure directory exists
	dir := "scenarios"
	// Try relative to working dir, also try parent of schema path
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// try finding project root via schema path
		_ = os.MkdirAll(dir, 0755)
	}
	filename := filepath.Join(dir, fmt.Sprintf("generated_%s.json", sanitizeFilename(id)))
	// Don't overwrite if exists, append timestamp
	if _, err := os.Stat(filename); err == nil {
		filename = filepath.Join(dir, fmt.Sprintf("generated_%s_%d.json", sanitizeFilename(id), time.Now().Unix()))
	}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, " ", "_")
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

func hasPlotError(details []string) bool {
	for _, d := range details {
		if strings.Contains(d, "plot") && (strings.Contains(d, "not found") || strings.Contains(d, "not inside")) {
			return true
		}
		if strings.Contains(d, "clearing") && strings.Contains(d, "not found") {
			return true
		}
	}
	return false
}

func truncateErr(s string) string {
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}
