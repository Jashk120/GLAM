package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaClientGenerateUsesLocalChatAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("path = %q, want /api/chat", r.URL.Path)
		}
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request["model"] != "qwen2.5-coder:14b" {
			t.Fatalf("model = %v", request["model"])
		}
		if request["stream"] != false || request["format"] != "json" {
			t.Fatalf("expected non-streaming JSON request, got %#v", request)
		}
		messages, ok := request["messages"].([]interface{})
		if !ok || len(messages) != 2 {
			t.Fatalf("messages = %#v", request["messages"])
		}
		_, _ = w.Write([]byte("{\"message\":{\"role\":\"assistant\",\"content\":\"```json\\n{\\\"id\\\":\\\"local-game\\\"}\\n```\"},\"done\":true}"))
	}))
	defer server.Close()

	client := &OllamaClient{Endpoint: server.URL + "/api/chat", Model: "qwen2.5-coder:14b", HTTPClient: server.Client()}
	got, err := client.Generate("make a game", []byte(`{"type":"object"}`), []byte(`[]`))
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if got != `{"id":"local-game"}` {
		t.Fatalf("Generate() = %q", got)
	}
}

func TestNewScenarioGeneratorSelectsOllama(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "ollama")
	t.Setenv("OLLAMA_MODEL", "qwen2.5-coder:14b")
	provider, err := NewScenarioGenerator()
	if err != nil {
		t.Fatalf("NewScenarioGenerator() error: %v", err)
	}
	if _, ok := provider.(*OllamaClient); !ok {
		t.Fatalf("provider = %T, want *OllamaClient", provider)
	}
}
