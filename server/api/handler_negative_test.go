package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"glam/server/llm"
)

// errorRoundTripper simulates LLM transport failure for 502 test.
type errorRoundTripper struct{}

func (e *errorRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, &fakeNetError{"simulated network failure"}
}

type fakeNetError struct{ msg string }

func (f *fakeNetError) Error() string   { return f.msg }
func (f *fakeNetError) Timeout() bool   { return false }
func (f *fakeNetError) Temporary() bool { return false }

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	cands := []string{"../../schema/scenario.schema.json", "../schema/scenario.schema.json", "schema/scenario.schema.json"}
	var sp, rp string
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			if abs, err := filepath.Abs(c); err == nil {
				sp = abs
				rp = filepath.Join(filepath.Dir(abs), "asset-registry.json")
			} else {
				sp = c
				rp = filepath.Join(filepath.Dir(c), "asset-registry.json")
			}
			break
		}
	}
	if sp == "" {
		t.Fatal("cannot find schema")
	}
	sj, err := os.ReadFile(sp)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	rj, err := os.ReadFile(rp)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	return &Handler{SchemaPath: sp, RegistryPath: rp, SchemaJSON: sj, RegistryJSON: rj, LLM: &llm.OpenRouterClient{APIKey: ""}}
}

func decodeErrResp(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode error response: %v body=%s", err, rec.Body.String())
	}
	return m
}

func withCORSTest(next http.Handler) http.Handler {
	allow := map[string]struct{}{"http://localhost:5173": {}, "http://127.0.0.1:5173": {}, "http://localhost:3000": {}, "http://127.0.0.1:3000": {}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := r.Header.Get("Origin")
		if _, ok := allow[o]; ok && o != "" {
			w.Header().Set("Access-Control-Allow-Origin", o)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func TestHandleGenerateNegative(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		body       string
		apiKey     string
		mockLLM502 bool
		want       int
		contains   string
	}{
		{"GET→405", http.MethodGet, `{"prompt":"hi"}`, "", false, http.StatusMethodNotAllowed, "method not allowed"},
		{"invalid JSON", http.MethodPost, `{bad`, "", false, http.StatusBadRequest, "invalid JSON"},
		{"empty object prompt→400", http.MethodPost, `{}`, "", false, http.StatusBadRequest, "prompt must be"},
		{"empty string prompt→400", http.MethodPost, `{"prompt":""}`, "", false, http.StatusBadRequest, "prompt must be"},
		{"whitespace prompt→400", http.MethodPost, `{"prompt":"   "}`, "", false, http.StatusBadRequest, "prompt must be"},
		{"prompt 2001 chars→400", http.MethodPost, `{"prompt":"` + strings.Repeat("a", 2001) + `"}`, "", false, http.StatusBadRequest, "prompt must be"},
		{"missing API key→500", http.MethodPost, `{"prompt":"hello"}`, "", false, http.StatusInternalServerError, "OPENROUTER_API_KEY"},
		{"OPTIONS→204", http.MethodOptions, ``, "", false, http.StatusNoContent, ""},
		{"LLM failure→502", http.MethodPost, `{"prompt":"hello"}`, "test-key", true, http.StatusBadGateway, "LLM request failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t)
			if tc.apiKey != "" {
				h.LLM.APIKey = tc.apiKey
			}
			if tc.mockLLM502 {
				h.LLM.Endpoint = "http://127.0.0.1:1"
				h.LLM.HTTPClient = &http.Client{Transport: &errorRoundTripper{}}
			}
			t.Setenv("GLAM_ROOT", t.TempDir())
			req := httptest.NewRequest(tc.method, "/api/scenario/generate", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.HandleGenerate(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("want %d got %d body=%s", tc.want, rec.Code, rec.Body.String())
			}
			if tc.contains != "" && !strings.Contains(rec.Body.String(), tc.contains) {
				t.Fatalf("body missing %q: %s", tc.contains, rec.Body.String())
			}
			if tc.want != http.StatusNoContent && rec.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("expected json content-type got %q", rec.Header().Get("Content-Type"))
			}
		})
	}
}

func TestHandleGenerateCORS(t *testing.T) {
	h := newTestHandler(t)
	wrapped := withCORSTest(http.HandlerFunc(h.HandleGenerate))
	req := httptest.NewRequest(http.MethodPost, "/api/scenario/generate", strings.NewReader(`{"prompt":"hi"}`))
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("expected CORS for allowed origin got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/scenario/generate", strings.NewReader(`{"prompt":"hi"}`))
	req2.Header.Set("Origin", "http://evil.com")
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)
	if rec2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("disallowed must not have CORS header got %q", rec2.Header().Get("Access-Control-Allow-Origin"))
	}
	req3 := httptest.NewRequest(http.MethodOptions, "/api/scenario/generate", nil)
	req3.Header.Set("Origin", "http://localhost:5173")
	rec3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusNoContent {
		t.Fatalf("preflight want 204 got %d", rec3.Code)
	}
}

type successRoundTripper struct {
	body string
}

func (s *successRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	respBody := `{"choices":[{"message":{"content":` + s.body + `}}]}`
	// s.body already JSON-escaped? we need to marshal scenario as JSON string inside content
	// To keep it simple, expect s.body is already JSON-marshaled scenario string escaped via json.Marshal
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(respBody)),
		Header:     make(http.Header),
	}, nil
}

func scenarioWithExtraFieldJSON() string {
	m := map[string]interface{}{
		"id": "test-warnings", "title": "Warnings Test", "version": "1.0",
		"world": map[string]interface{}{"template": "desert", "spawn": map[string]interface{}{"x": 2, "y": 2}, "size": map[string]interface{}{"cols": 15, "rows": 12}},
		"characters": []interface{}{
			map[string]interface{}{"id": "char_a", "name": "Alice", "position": map[string]interface{}{"x": 1, "y": 1}, "extraField": "oops"},
		},
		"buildings": []interface{}{
			map[string]interface{}{"id": "bld_a", "typeAssetId": "shop_small", "position": map[string]interface{}{"x": 3, "y": 3}},
		},
		"objects": []interface{}{
			map[string]interface{}{"id": "obj_a", "assetId": "object_chest", "position": map[string]interface{}{"x": 4, "y": 4}},
		},
		"missions": []interface{}{
			map[string]interface{}{"id": "mission_a", "title": "Talk", "description": "Talk to Alice", "trigger": map[string]interface{}{"entityId": "char_a"}, "reward": 100},
		},
	}
	b, _ := json.Marshal(m)
	// content must be a JSON string value, so we need to JSON-escape the scenario string
	escaped, _ := json.Marshal(string(b))
	return string(escaped)
}

func scenarioWithPlotMismatchJSON() string {
	m := map[string]interface{}{
		"id": "test-plot-fix", "title": "Plot Fix Test", "version": "1.0",
		"world": map[string]interface{}{"template": "town", "spawn": map[string]interface{}{"x": 7, "y": 9}, "size": map[string]interface{}{"cols": 15, "rows": 12}},
		"characters": []interface{}{
			map[string]interface{}{"id": "char_a", "name": "Alice", "position": map[string]interface{}{"x": 2, "y": 2}, "plot": "clearing_2"},
		},
		"buildings": []interface{}{
			map[string]interface{}{"id": "bld_a", "typeAssetId": "shop_small", "position": map[string]interface{}{"x": 7, "y": 2}, "plot": "clearing_1"},
		},
		"objects": []interface{}{
			map[string]interface{}{"id": "obj_a", "assetId": "object_chest", "position": map[string]interface{}{"x": 12, "y": 10}},
		},
		"missions": []interface{}{
			map[string]interface{}{"id": "mission_a", "title": "Talk", "description": "Talk to Alice", "trigger": map[string]interface{}{"entityId": "char_a"}},
		},
	}
	b, _ := json.Marshal(m)
	escaped, _ := json.Marshal(string(b))
	return string(escaped)
}

func TestHandleGenerate_Warnings(t *testing.T) {
	t.Run("extra field auto-repair returns warnings", func(t *testing.T) {
		h := newTestHandler(t)
		h.LLM.APIKey = "test-key"
		h.LLM.HTTPClient = &http.Client{Transport: &successRoundTripper{body: scenarioWithExtraFieldJSON()}}
		t.Setenv("GLAM_ROOT", t.TempDir())
		req := httptest.NewRequest(http.MethodPost, "/api/scenario/generate", strings.NewReader(`{"prompt":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.HandleGenerate(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode resp: %v", err)
		}
		if _, ok := resp["scenario"]; !ok {
			t.Fatalf("missing scenario in %s", rec.Body.String())
		}
		w, ok := resp["warnings"]
		if !ok {
			t.Fatalf("expected warnings field when auto-repair happened, got %s", rec.Body.String())
		}
		arr, ok := w.([]interface{})
		if !ok || len(arr) == 0 {
			t.Fatalf("warnings not array or empty: %v", w)
		}
		joined := strings.ToLower(strings.Join(func() []string {
			var s []string
			for _, v := range arr {
				if str, ok := v.(string); ok {
					s = append(s, str)
				}
			}
			return s
		}(), " | "))
		if !strings.Contains(joined, "additionalproperties") && !strings.Contains(joined, "stripped") {
			t.Fatalf("warnings should mention additionalProperties/stripped, got %v", arr)
		}
	})
	t.Run("plot mismatch auto-repair returns warnings", func(t *testing.T) {
		h := newTestHandler(t)
		h.LLM.APIKey = "test-key"
		h.LLM.HTTPClient = &http.Client{Transport: &successRoundTripper{body: scenarioWithPlotMismatchJSON()}}
		t.Setenv("GLAM_ROOT", t.TempDir())
		req := httptest.NewRequest(http.MethodPost, "/api/scenario/generate", strings.NewReader(`{"prompt":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.HandleGenerate(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode resp: %v", err)
		}
		if _, ok := resp["scenario"]; !ok {
			t.Fatalf("missing scenario in %s", rec.Body.String())
		}
		w, ok := resp["warnings"]
		if !ok {
			t.Fatalf("expected warnings field for plot fix, got %s", rec.Body.String())
		}
		arr, ok := w.([]interface{})
		if !ok || len(arr) == 0 {
			t.Fatalf("warnings not array or empty: %v", w)
		}
		joined := strings.ToLower(strings.Join(func() []string {
			var s []string
			for _, v := range arr {
				if str, ok := v.(string); ok {
					s = append(s, str)
				}
			}
			return s
		}(), " | "))
		if !strings.Contains(joined, "auto-fixed") && !strings.Contains(joined, "plot") {
			t.Fatalf("warnings should mention auto-fixed/plot, got %v", arr)
		}
	})
	t.Run("no warnings when no auto-repair", func(t *testing.T) {
		// Valid scenario should not produce warnings field
		m := map[string]interface{}{
			"id": "clean-scenario", "title": "Clean", "version": "1.0",
			"world":      map[string]interface{}{"template": "desert", "spawn": map[string]interface{}{"x": 2, "y": 2}, "size": map[string]interface{}{"cols": 15, "rows": 12}},
			"characters": []interface{}{map[string]interface{}{"id": "char_a", "name": "Alice", "position": map[string]interface{}{"x": 1, "y": 1}}},
			"buildings":  []interface{}{map[string]interface{}{"id": "bld_a", "typeAssetId": "shop_small", "position": map[string]interface{}{"x": 3, "y": 3}}},
			"objects":    []interface{}{map[string]interface{}{"id": "obj_a", "assetId": "object_chest", "position": map[string]interface{}{"x": 4, "y": 4}}},
			"missions":   []interface{}{map[string]interface{}{"id": "mission_a", "title": "Talk", "description": "Talk to Alice", "trigger": map[string]interface{}{"entityId": "char_a"}}},
		}
		b, _ := json.Marshal(m)
		escaped, _ := json.Marshal(string(b))
		h := newTestHandler(t)
		h.LLM.APIKey = "test-key"
		h.LLM.HTTPClient = &http.Client{Transport: &successRoundTripper{body: string(escaped)}}
		t.Setenv("GLAM_ROOT", t.TempDir())
		req := httptest.NewRequest(http.MethodPost, "/api/scenario/generate", strings.NewReader(`{"prompt":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.HandleGenerate(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode resp: %v", err)
		}
		if _, ok := resp["warnings"]; ok {
			t.Fatalf("unexpected warnings for clean scenario: %v", resp["warnings"])
		}
	})
}

func TestHandleValidateNegative(t *testing.T) {
	cases := []struct {
		name   string
		method string
		body   string
		want   int
		substr string
	}{
		{"invalid JSON", http.MethodPost, `not-json`, http.StatusBadRequest, "validation failed"},
		{"schema violation missing required", http.MethodPost, `{"id":"bad"}`, http.StatusBadRequest, "validation failed"},
		{"forbidden field code", http.MethodPost, `{"id":"t1","title":"t","world":{"template":"town","spawn":{"x":1,"y":1},"size":{"cols":8,"rows":8}},"characters":[],"buildings":[],"objects":[],"missions":[],"code":"evil"}`, http.StatusBadRequest, "forbidden"},
		{"method not allowed PUT", http.MethodPut, `{}`, http.StatusMethodNotAllowed, "method not allowed"},
		{"empty body", http.MethodPost, ``, http.StatusBadRequest, "read body failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t)
			req := httptest.NewRequest(tc.method, "/api/scenario/validate", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.HandleValidate(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("want %d got %d body=%s", tc.want, rec.Code, rec.Body.String())
			}
			if tc.substr != "" && !strings.Contains(strings.ToLower(rec.Body.String()), strings.ToLower(tc.substr)) {
				t.Fatalf("want %q got %s", tc.substr, rec.Body.String())
			}
			m := decodeErrResp(t, rec)
			if _, ok := m["error"]; !ok {
				t.Fatalf("error field missing in %s", rec.Body.String())
			}
		})
	}
	t.Run("OPTIONS→204", func(t *testing.T) {
		h := newTestHandler(t)
		req := httptest.NewRequest(http.MethodOptions, "/api/scenario/validate", nil)
		rec := httptest.NewRecorder()
		h.HandleValidate(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("want 204 got %d", rec.Code)
		}
	})
}
