package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type repairingGenerator struct {
	repairCalls int
}

func (g *repairingGenerator) IsConfigured() bool { return true }

func (g *repairingGenerator) Generate(_ string, _ []byte, _ []byte) (string, error) {
	return `{"id":"broken"}`, nil
}

func (g *repairingGenerator) Repair(_ string, details []string, _ []byte, _ []byte) (string, error) {
	g.repairCalls++
	if len(details) == 0 {
		return "", nil
	}
	return `{"id":"repaired","title":"Repaired","world":{"template":"school","spawn":{"x":1,"y":1},"size":{"cols":8,"rows":8}},"characters":[],"buildings":[],"objects":[],"missions":[]}`, nil
}

func TestHandleGenerateRepairsLocalProviderOutput(t *testing.T) {
	h := newTestHandler(t)
	generator := &repairingGenerator{}
	h.Generator = generator
	req := httptest.NewRequest(http.MethodPost, "/api/scenario/generate", strings.NewReader(`{"prompt":"make a lesson"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleGenerate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if generator.repairCalls != 1 {
		t.Fatalf("repair calls = %d, want 1", generator.repairCalls)
	}
}
