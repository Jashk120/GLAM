package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"glam/server/agent"
	"glam/server/llm"
	"glam/server/pipeline"
	"glam/server/session"
)

type ChatHandler struct {
	Handler      *Handler
	SessionStore *session.Store
}

func NewChatHandler(h *Handler, store *session.Store) *ChatHandler {
	if store == nil {
		store = session.NewStore()
	}
	return &ChatHandler{Handler: h, SessionStore: store}
}

func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("sess_%d", len(b))
	}
	return hex.EncodeToString(b)
}

func (ch *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", []string{err.Error()})
		return
	}
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		writeError(w, http.StatusBadRequest, "message must be non-empty", nil)
		return
	}
	if len(msg) > 2000 {
		writeError(w, http.StatusBadRequest, "message too long (max 2000)", nil)
		return
	}

	sid := strings.TrimSpace(req.SessionID)
	if sid == "" {
		sid = generateSessionID()
	}

	if ch.Handler.LLM.APIKey == "" {
		writeError(w, http.StatusInternalServerError, "server not configured: OPENROUTER_API_KEY missing", nil)
		return
	}

	history := ch.SessionStore.Get(sid)
	if len(history) == 0 {
		history = []llm.Message{{Role: "system", Content: agent.SystemPrompt}}
	}

	history = append(history, llm.NewUserMessage(msg))

	deps := pipeline.Deps{
		SchemaJSON:   ch.Handler.SchemaJSON,
		RegistryJSON: ch.Handler.RegistryJSON,
		SchemaPath:   ch.Handler.SchemaPath,
		RegistryPath: ch.Handler.RegistryPath,
	}

	updated, err := agent.RunTurn(r.Context(), ch.Handler.LLM, history, deps)
	if err != nil {
		// If max iterations or LLM error, return what we have with error
		writeError(w, http.StatusBadGateway, "agent turn failed", []string{err.Error()})
		// Still persist? Don't persist on hard failure to avoid corruption
		return
	}

	ch.SessionStore.Set(sid, updated)

	// Extract last assistant reply and possible scenario
	reply := ""
	var scenario interface{}
	// Walk backwards to find last assistant message
	for i := len(updated) - 1; i >= 0; i-- {
		m := updated[i]
		if m.Role == "assistant" && m.Content != "" {
			reply = m.Content
			break
		}
	}
	// Also check for scenario in tool results (last tool result containing scenario)
	for i := len(updated) - 1; i >= 0; i-- {
		m := updated[i]
		if m.Role == "tool" && strings.Contains(m.Content, "\"scenario\"") {
			var parsed map[string]interface{}
			if json.Unmarshal([]byte(m.Content), &parsed) == nil {
				if sc, ok := parsed["scenario"]; ok {
					scenario = sc
					if reply == "" {
						reply = "Scenario generated successfully."
					}
					break
				}
			}
		}
	}
	// Detect ask_teacher pause: last tool result contains question and reply empty
	if scenario == nil && reply == "" {
		for i := len(updated) - 1; i >= 0; i-- {
			m := updated[i]
			if m.Role == "tool" && strings.Contains(m.Content, "\"question\"") {
				var parsed map[string]interface{}
				if json.Unmarshal([]byte(m.Content), &parsed) == nil {
					if q, ok := parsed["question"].(string); ok && q != "" {
						reply = q
						break
					}
				}
			}
			if m.Role == "assistant" && m.Content != "" {
				reply = m.Content
				break
			}
		}
	}
	if reply == "" && scenario == nil {
		reply = "I'm ready to help. Tell me the topic and age group."
	}

	// Optionally save scenario to disk if present
	if scenario != nil {
		if scMap, ok := scenario.(map[string]interface{}); ok {
			_ = saveGenerated(scMap)
		}
	}

	resp := map[string]interface{}{
		"session_id": sid,
		"reply":      reply,
	}
	if scenario != nil {
		resp["scenario"] = scenario
	}

	writeJSON(w, http.StatusOK, resp)
}
