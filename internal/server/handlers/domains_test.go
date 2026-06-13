package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/davidolrik/corto/internal/server/handlers"
)

type memoryDomainStore struct {
	mu      sync.RWMutex
	domains map[string]*handlers.DomainData
	seq     int
}

func newMemoryDomainStore() *memoryDomainStore {
	return &memoryDomainStore{
		domains: make(map[string]*handlers.DomainData),
	}
}

func (s *memoryDomainStore) CreateDomain(_ context.Context, d *handlers.DomainData) (*handlers.DomainData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check for duplicate FQDN
	for _, existing := range s.domains {
		if existing.FQDN == d.FQDN {
			return nil, fmt.Errorf("domain %q already exists: %w", d.FQDN, handlers.ErrConflict)
		}
	}
	s.seq++
	d.PublicID = fmt.Sprintf("dom_%d", s.seq)
	now := time.Now().Truncate(time.Second)
	d.CreatedAt = now
	d.UpdatedAt = now
	s.domains[d.PublicID] = d
	return d, nil
}

func (s *memoryDomainStore) GetDomain(_ context.Context, publicID string) (*handlers.DomainData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.domains[publicID]
	if !ok {
		return nil, fmt.Errorf("domain %q %w", publicID, handlers.ErrNotFound)
	}
	return d, nil
}

func (s *memoryDomainStore) ListDomains(_ context.Context) ([]*handlers.DomainData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*handlers.DomainData, 0, len(s.domains))
	for _, d := range s.domains {
		result = append(result, d)
	}
	return result, nil
}

func (s *memoryDomainStore) UpdateDomain(_ context.Context, publicID string, d *handlers.DomainData) (*handlers.DomainData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.domains[publicID]
	if !ok {
		return nil, fmt.Errorf("domain %q %w", publicID, handlers.ErrNotFound)
	}
	d.PublicID = publicID
	d.CreatedAt = existing.CreatedAt
	d.UpdatedAt = time.Now().Truncate(time.Second)
	s.domains[publicID] = d
	return d, nil
}

func (s *memoryDomainStore) PatchDomain(_ context.Context, publicID string, patch *handlers.DomainPatch) (*handlers.DomainData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.domains[publicID]
	if !ok {
		return nil, fmt.Errorf("domain %q %w", publicID, handlers.ErrNotFound)
	}
	if patch.FQDN != nil {
		existing.FQDN = *patch.FQDN
	}
	if patch.FallbackURL != nil {
		existing.FallbackURL = *patch.FallbackURL
	}
	if patch.Description != nil {
		existing.Description = *patch.Description
	}
	existing.UpdatedAt = time.Now().Truncate(time.Second)
	return existing, nil
}

func (s *memoryDomainStore) DeleteDomain(_ context.Context, publicID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[publicID]; !ok {
		return fmt.Errorf("domain %q %w", publicID, handlers.ErrNotFound)
	}
	delete(s.domains, publicID)
	return nil
}

func TestCreateDomainDuplicate(t *testing.T) {
	api := setupDomainAPI(t)

	api.Post("/api/domains", map[string]any{"fqdn": "go.example.com"})
	resp := api.Post("/api/domains", map[string]any{"fqdn": "go.example.com"})

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, resp.Code, resp.Body.String())
	}
}

func TestDomainDescriptionRoundTrip(t *testing.T) {
	api := setupDomainAPI(t)

	resp := api.Post("/api/domains", map[string]any{
		"fqdn":        "go.example.com",
		"description": "Primary short link domain",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var created handlers.DomainBody
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding create response: %v", err)
	}
	if created.Description != "Primary short link domain" {
		t.Errorf("expected description %q, got %q", "Primary short link domain", created.Description)
	}

	patchResp := api.Patch("/api/domains/"+created.PublicID, map[string]any{
		"description": "Updated description",
	})
	if patchResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, patchResp.Code, patchResp.Body.String())
	}

	var patched handlers.DomainBody
	if err := json.Unmarshal(patchResp.Body.Bytes(), &patched); err != nil {
		t.Fatalf("decoding patch response: %v", err)
	}
	if patched.Description != "Updated description" {
		t.Errorf("expected patched description %q, got %q", "Updated description", patched.Description)
	}
	if patched.FQDN != "go.example.com" {
		t.Errorf("expected fqdn to be unchanged, got %q", patched.FQDN)
	}
}

func setupDomainAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	store := newMemoryDomainStore()
	handlers.RegisterDomainRoutes(api, store)
	return api
}

func TestCreateDomain(t *testing.T) {
	api := setupDomainAPI(t)

	resp := api.Post("/api/domains", map[string]any{
		"fqdn":         "short.io",
		"fallback_url": "https://example.com",
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var body handlers.DomainBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if body.PublicID == "" {
		t.Fatal("expected public_id to be set")
	}
	if body.FQDN != "short.io" {
		t.Errorf("expected fqdn %q, got %q", "short.io", body.FQDN)
	}
	if body.FallbackURL != "https://example.com" {
		t.Errorf("expected fallback_url %q, got %q", "https://example.com", body.FallbackURL)
	}
	if body.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestCreateDomainWithoutFallback(t *testing.T) {
	api := setupDomainAPI(t)

	resp := api.Post("/api/domains", map[string]any{
		"fqdn": "minimal.io",
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var body handlers.DomainBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.FQDN != "minimal.io" {
		t.Errorf("expected fqdn %q, got %q", "minimal.io", body.FQDN)
	}
	if body.FallbackURL != "" {
		t.Errorf("expected empty fallback_url, got %q", body.FallbackURL)
	}
}

func TestCreateDomainMissingRequired(t *testing.T) {
	api := setupDomainAPI(t)

	resp := api.Post("/api/domains", map[string]any{
		"fallback_url": "https://example.com",
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}
}

func TestGetDomain(t *testing.T) {
	api := setupDomainAPI(t)

	createResp := api.Post("/api/domains", map[string]any{
		"fqdn":         "short.io",
		"fallback_url": "https://fallback.com",
	})
	var created handlers.DomainBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Get("/api/domains/" + created.PublicID)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.DomainBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.PublicID != created.PublicID {
		t.Errorf("expected public_id %q, got %q", created.PublicID, body.PublicID)
	}
	if body.FQDN != "short.io" {
		t.Errorf("expected fqdn %q, got %q", "short.io", body.FQDN)
	}
	if body.FallbackURL != "https://fallback.com" {
		t.Errorf("expected fallback_url %q, got %q", "https://fallback.com", body.FallbackURL)
	}
}

func TestGetDomainNotFound(t *testing.T) {
	api := setupDomainAPI(t)

	resp := api.Get("/api/domains/nonexistent")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestListDomains(t *testing.T) {
	api := setupDomainAPI(t)

	// Empty list
	resp := api.Get("/api/domains")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	var empty []handlers.DomainBody
	json.Unmarshal(resp.Body.Bytes(), &empty)
	if len(empty) != 0 {
		t.Errorf("expected 0 domains, got %d", len(empty))
	}

	// Create two
	api.Post("/api/domains", map[string]any{"fqdn": "a.io"})
	api.Post("/api/domains", map[string]any{"fqdn": "b.io"})

	resp = api.Get("/api/domains")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	var domains []handlers.DomainBody
	json.Unmarshal(resp.Body.Bytes(), &domains)
	if len(domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(domains))
	}
}

func TestUpdateDomain(t *testing.T) {
	api := setupDomainAPI(t)

	createResp := api.Post("/api/domains", map[string]any{
		"fqdn":         "old.io",
		"fallback_url": "https://old.com",
	})
	var created handlers.DomainBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Put("/api/domains/"+created.PublicID, map[string]any{
		"fqdn":         "replaced.io",
		"fallback_url": "https://replaced.com",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.DomainBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.PublicID != created.PublicID {
		t.Errorf("expected public_id %q, got %q", created.PublicID, body.PublicID)
	}
	if body.FQDN != "replaced.io" {
		t.Errorf("expected fqdn %q, got %q", "replaced.io", body.FQDN)
	}
	if body.FallbackURL != "https://replaced.com" {
		t.Errorf("expected fallback_url %q, got %q", "https://replaced.com", body.FallbackURL)
	}
	if body.CreatedAt != created.CreatedAt {
		t.Error("expected created_at to be preserved")
	}
}

func TestUpdateDomainNotFound(t *testing.T) {
	api := setupDomainAPI(t)

	resp := api.Put("/api/domains/nonexistent", map[string]any{
		"fqdn": "x.io",
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestPatchDomainFQDN(t *testing.T) {
	api := setupDomainAPI(t)

	createResp := api.Post("/api/domains", map[string]any{
		"fqdn":         "original.io",
		"fallback_url": "https://keep-this.com",
	})
	var created handlers.DomainBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	// Patch only FQDN
	resp := api.Patch("/api/domains/"+created.PublicID, map[string]any{
		"fqdn": "patched.io",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.DomainBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.FQDN != "patched.io" {
		t.Errorf("expected fqdn %q, got %q", "patched.io", body.FQDN)
	}
	if body.FallbackURL != "https://keep-this.com" {
		t.Errorf("expected fallback_url %q to be unchanged, got %q", "https://keep-this.com", body.FallbackURL)
	}
}

func TestPatchDomainFallbackURL(t *testing.T) {
	api := setupDomainAPI(t)

	createResp := api.Post("/api/domains", map[string]any{
		"fqdn":         "keep-this.io",
		"fallback_url": "https://old.com",
	})
	var created handlers.DomainBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	// Patch only fallback_url
	resp := api.Patch("/api/domains/"+created.PublicID, map[string]any{
		"fallback_url": "https://patched.com",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.DomainBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.FQDN != "keep-this.io" {
		t.Errorf("expected fqdn %q to be unchanged, got %q", "keep-this.io", body.FQDN)
	}
	if body.FallbackURL != "https://patched.com" {
		t.Errorf("expected fallback_url %q, got %q", "https://patched.com", body.FallbackURL)
	}
}

func TestPatchDomainNotFound(t *testing.T) {
	api := setupDomainAPI(t)

	resp := api.Patch("/api/domains/nonexistent", map[string]any{
		"fqdn": "nope.io",
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestDeleteDomain(t *testing.T) {
	api := setupDomainAPI(t)

	createResp := api.Post("/api/domains", map[string]any{
		"fqdn": "bye.io",
	})
	var created handlers.DomainBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Delete("/api/domains/" + created.PublicID)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, resp.Code, resp.Body.String())
	}

	// Verify it's gone
	getResp := api.Get("/api/domains/" + created.PublicID)
	if getResp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d after delete, got %d", http.StatusNotFound, getResp.Code)
	}
}

func TestDeleteDomainNotFound(t *testing.T) {
	api := setupDomainAPI(t)

	resp := api.Delete("/api/domains/nonexistent")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}
