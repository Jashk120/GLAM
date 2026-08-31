package api

import (
	"encoding/json"
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
