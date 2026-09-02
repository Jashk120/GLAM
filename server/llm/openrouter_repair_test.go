package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenRouterClientRepairSendsValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		messages := request["messages"].([]interface{})
		user := messages[1].(map[string]interface{})["content"].(string)
		if user == "" || request["response_format"] == nil {
			t.Fatalf("repair request missing JSON instructions: %#v", request)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"id\":\"repaired\"}"}}]}`))
	}))
	defer server.Close()

	client := &OpenRouterClient{APIKey: "test-key", Endpoint: server.URL, Model: "test-model", HTTPClient: server.Client()}
	got, err := client.Repair(`{"id":"bad"}`, []string{"missing title"}, []byte(`{"type":"object"}`), []byte(`[]`))
	if err != nil {
		t.Fatalf("Repair() error: %v", err)
	}
	if got != `{"id":"repaired"}` {
		t.Fatalf("Repair() = %q", got)
	}
}
