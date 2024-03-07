package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/davidolrik/corto/internal/server/handlers"
)

func TestGetVersion(t *testing.T) {
	_, api := humatest.New(t)
	handlers.RegisterVersionRoutes(api, "1.2.3")

	resp := api.Get("/api/version")

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Version != "1.2.3" {
		t.Errorf("expected version %q, got %q", "1.2.3", body.Version)
	}
}

func TestGetVersionFallsBackToDevel(t *testing.T) {
	_, api := humatest.New(t)
	handlers.RegisterVersionRoutes(api, "")

	resp := api.Get("/api/version")

	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Version != "devel" {
		t.Errorf("expected version %q, got %q", "devel", body.Version)
	}
}
