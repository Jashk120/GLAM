package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"glam/server/llm"
)

func TestHandleAssetsNegative(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/assets", nil)
	rec := httptest.NewRecorder()
	h.HandleAssets(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST assets want 405 got %d", rec.Code)
	}
	req2 := httptest.NewRequest(http.MethodOptions, "/api/assets", nil)
	rec2 := httptest.NewRecorder()
	h.HandleAssets(rec2, req2)
	if rec2.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS want 204 got %d", rec2.Code)
	}
	req3 := httptest.NewRequest(http.MethodGet, "/api/assets", nil)
	rec3 := httptest.NewRecorder()
	h.HandleAssets(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("GET assets want 200 got %d", rec3.Code)
	}
	var arr []interface{}
	if err := json.Unmarshal(rec3.Body.Bytes(), &arr); err != nil {
		t.Fatalf("assets not array: %v body=%s", err, rec3.Body.String())
	}
	if len(arr) != 21 {
		t.Fatalf("want 21 assets got %d", len(arr))
	}
	wrapped := withCORSTest(http.HandlerFunc(h.HandleAssets))
	req4 := httptest.NewRequest(http.MethodGet, "/api/assets", nil)
	req4.Header.Set("Origin", "http://localhost:5173")
	rec4 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec4, req4)
	if rec4.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("CORS missing for assets")
	}
}

func TestHandleHealthAndScenarios(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	HandleHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health GET want 200 got %d", rec.Code)
	}
	var hm map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &hm); err != nil || hm["status"] != "ok" {
		t.Fatalf("health body bad: %s", rec.Body.String())
	}
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, fn := range []string{"scenario.schema.json", "asset-registry.json"} {
		data, _ := os.ReadFile(filepath.Join("../../schema", fn))
		if len(data) == 0 {
			data, _ = os.ReadFile(filepath.Join("../..", "schema", fn))
		}
		_ = os.WriteFile(filepath.Join(schemaDir, fn), data, 0644)
	}
	scenDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenDir, 0755); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(schemaDir, "scenario.schema.json")
	rp := filepath.Join(schemaDir, "asset-registry.json")
	sj, _ := os.ReadFile(sp)
	rj, _ := os.ReadFile(rp)
	h := &Handler{SchemaPath: sp, RegistryPath: rp, SchemaJSON: sj, RegistryJSON: rj, LLM: &llm.OpenRouterClient{APIKey: ""}}
	req2 := httptest.NewRequest(http.MethodGet, "/api/scenarios", nil)
	rec2 := httptest.NewRecorder()
	h.HandleListScenarios(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list empty want 200 got %d", rec2.Code)
	}
	var lr map[string]interface{}
	if err := json.Unmarshal(rec2.Body.Bytes(), &lr); err != nil {
		t.Fatalf("list decode: %v", err)
	}
	if arr, _ := lr["scenarios"].([]interface{}); len(arr) != 0 {
		t.Fatalf("empty dir want 0 got %d", len(arr))
	}
	_ = os.WriteFile(filepath.Join(scenDir, "bad.json"), []byte("{bad json"), 0644)
	_ = os.WriteFile(filepath.Join(scenDir, "good.json"), []byte(`{"id":"good","title":"Good","world":{"template":"town","spawn":{"x":1,"y":1},"size":{"cols":8,"rows":8}},"characters":[],"buildings":[],"objects":[],"missions":[]}`), 0644)
	_ = os.WriteFile(filepath.Join(scenDir, "note.txt"), []byte("ignore"), 0644)
	req3 := httptest.NewRequest(http.MethodGet, "/api/scenarios", nil)
	rec3 := httptest.NewRecorder()
	h.HandleListScenarios(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("list want 200 got %d", rec3.Code)
	}
	var lr2 map[string]interface{}
	_ = json.Unmarshal(rec3.Body.Bytes(), &lr2)
	if arr, _ := lr2["scenarios"].([]interface{}); len(arr) != 1 {
		t.Fatalf("want 1 valid scenario got %d body=%s", len(arr), rec3.Body.String())
	}
	req4 := httptest.NewRequest(http.MethodPost, "/api/scenarios", nil)
	rec4 := httptest.NewRecorder()
	h.HandleListScenarios(rec4, req4)
	if rec4.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST list want 405 got %d", rec4.Code)
	}
	req5 := httptest.NewRequest(http.MethodGet, "/api/scenarios/unknown_id_xyz", nil)
	rec5 := httptest.NewRecorder()
	h.HandleGetScenario(rec5, req5)
	if rec5.Code != http.StatusNotFound {
		t.Fatalf("get unknown want 404 got %d body=%s", rec5.Code, rec5.Body.String())
	}
	req6 := httptest.NewRequest(http.MethodGet, "/api/scenarios/", nil)
	rec6 := httptest.NewRecorder()
	h.HandleGetScenario(rec6, req6)
	if rec6.Code != http.StatusBadRequest {
		t.Fatalf("missing id want 400 got %d", rec6.Code)
	}
	req7 := httptest.NewRequest(http.MethodPost, "/api/scenarios/good", nil)
	rec7 := httptest.NewRecorder()
	h.HandleGetScenario(rec7, req7)
	if rec7.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST getScenario want 405 got %d", rec7.Code)
	}
}

func TestWriteErrorShape(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "oops", []string{"a", "b"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", rec.Code)
	}
	var m map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	if m["error"] != "oops" {
		t.Fatalf("error field wrong: %v", m)
	}
	if d, ok := m["details"].([]interface{}); !ok || len(d) != 2 {
		t.Fatalf("details want 2 got %v", m["details"])
	}
	rec2 := httptest.NewRecorder()
	writeError(rec2, http.StatusBadRequest, "oops2", nil)
	var m2 map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &m2)
	if _, ok := m2["details"]; ok {
		t.Fatalf("details should be omitted when empty, got %v", m2)
	}
	rec3 := httptest.NewRecorder()
	writeError(rec3, http.StatusBadRequest, "oops3", []string{})
	var m3 map[string]interface{}
	_ = json.Unmarshal(rec3.Body.Bytes(), &m3)
	if _, ok := m3["details"]; ok {
		t.Fatalf("empty details should be omitted")
	}
}
