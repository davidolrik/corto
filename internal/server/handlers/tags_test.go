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

type memoryTagStore struct {
	mu   sync.RWMutex
	tags map[string]*handlers.TagData
	seq  int
}

func newMemoryTagStore() *memoryTagStore {
	return &memoryTagStore{
		tags: make(map[string]*handlers.TagData),
	}
}

func (s *memoryTagStore) CreateTag(_ context.Context, t *handlers.TagData) (*handlers.TagData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	t.PublicID = fmt.Sprintf("tag_%d", s.seq)
	now := time.Now().Truncate(time.Second)
	t.CreatedAt = now
	t.UpdatedAt = now
	s.tags[t.PublicID] = t
	return t, nil
}

func (s *memoryTagStore) GetTag(_ context.Context, publicID string) (*handlers.TagData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tags[publicID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return t, nil
}

func (s *memoryTagStore) ListTags(_ context.Context) ([]*handlers.TagData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*handlers.TagData, 0, len(s.tags))
	for _, t := range s.tags {
		result = append(result, t)
	}
	return result, nil
}

func (s *memoryTagStore) UpdateTag(_ context.Context, publicID string, t *handlers.TagData) (*handlers.TagData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tags[publicID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	t.PublicID = publicID
	t.CreatedAt = existing.CreatedAt
	t.UpdatedAt = time.Now().Truncate(time.Second)
	s.tags[publicID] = t
	return t, nil
}

func (s *memoryTagStore) PatchTag(_ context.Context, publicID string, patch *handlers.TagPatch) (*handlers.TagData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tags[publicID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	if patch.Slug != nil {
		existing.Slug = *patch.Slug
	}
	if patch.Color != nil {
		existing.Color = *patch.Color
	}
	if patch.Description != nil {
		existing.Description = *patch.Description
	}
	existing.UpdatedAt = time.Now().Truncate(time.Second)
	return existing, nil
}

func (s *memoryTagStore) DeleteTag(_ context.Context, publicID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tags[publicID]; !ok {
		return fmt.Errorf("not found")
	}
	delete(s.tags, publicID)
	return nil
}

func setupTagAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	store := newMemoryTagStore()
	handlers.RegisterTagRoutes(api, store)
	return api
}

func TestCreateTagWithColorAndDescription(t *testing.T) {
	api := setupTagAPI(t)

	resp := api.Post("/api/tags", map[string]any{
		"slug":        "campaigns",
		"color":       "#ff6600",
		"description": "Marketing campaign links",
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var body handlers.TagBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Color != "#ff6600" {
		t.Errorf("expected color %q, got %q", "#ff6600", body.Color)
	}
	if body.Description != "Marketing campaign links" {
		t.Errorf("expected description %q, got %q", "Marketing campaign links", body.Description)
	}
}

func TestCreateTagRejectsInvalidColor(t *testing.T) {
	api := setupTagAPI(t)

	resp := api.Post("/api/tags", map[string]any{
		"slug":  "campaigns",
		"color": "orange",
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}
}

func TestPatchTagColor(t *testing.T) {
	api := setupTagAPI(t)

	createResp := api.Post("/api/tags", map[string]any{"slug": "campaigns", "color": "#ff6600"})
	var created handlers.TagBody
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding create response: %v", err)
	}

	resp := api.Patch("/api/tags/"+created.PublicID, map[string]any{"color": "#00aa66"})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.TagBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Color != "#00aa66" {
		t.Errorf("expected color %q, got %q", "#00aa66", body.Color)
	}
	if body.Slug != "campaigns" {
		t.Errorf("expected slug to be unchanged, got %q", body.Slug)
	}
}

func TestCreateTag(t *testing.T) {
	api := setupTagAPI(t)

	resp := api.Post("/api/tags", map[string]any{
		"slug": "marketing",
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var body handlers.TagBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if body.PublicID == "" {
		t.Fatal("expected public_id to be set")
	}
	if body.Slug != "marketing" {
		t.Errorf("expected slug %q, got %q", "marketing", body.Slug)
	}
	if body.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestCreateTagMissingRequired(t *testing.T) {
	api := setupTagAPI(t)

	resp := api.Post("/api/tags", map[string]any{})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}
}

func TestGetTag(t *testing.T) {
	api := setupTagAPI(t)

	createResp := api.Post("/api/tags", map[string]any{"slug": "promo"})
	var created handlers.TagBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Get("/api/tags/" + created.PublicID)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.TagBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.PublicID != created.PublicID {
		t.Errorf("expected public_id %q, got %q", created.PublicID, body.PublicID)
	}
	if body.Slug != "promo" {
		t.Errorf("expected slug %q, got %q", "promo", body.Slug)
	}
}

func TestGetTagNotFound(t *testing.T) {
	api := setupTagAPI(t)

	resp := api.Get("/api/tags/nonexistent")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestListTags(t *testing.T) {
	api := setupTagAPI(t)

	// Empty list
	resp := api.Get("/api/tags")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	var empty []handlers.TagBody
	json.Unmarshal(resp.Body.Bytes(), &empty)
	if len(empty) != 0 {
		t.Errorf("expected 0 tags, got %d", len(empty))
	}

	// Create two
	api.Post("/api/tags", map[string]any{"slug": "alpha"})
	api.Post("/api/tags", map[string]any{"slug": "beta"})

	resp = api.Get("/api/tags")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	var tags []handlers.TagBody
	json.Unmarshal(resp.Body.Bytes(), &tags)
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestUpdateTag(t *testing.T) {
	api := setupTagAPI(t)

	createResp := api.Post("/api/tags", map[string]any{"slug": "old-slug"})
	var created handlers.TagBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Put("/api/tags/"+created.PublicID, map[string]any{
		"slug": "new-slug",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.TagBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.PublicID != created.PublicID {
		t.Errorf("expected public_id %q, got %q", created.PublicID, body.PublicID)
	}
	if body.Slug != "new-slug" {
		t.Errorf("expected slug %q, got %q", "new-slug", body.Slug)
	}
	if body.CreatedAt != created.CreatedAt {
		t.Error("expected created_at to be preserved")
	}
}

func TestUpdateTagNotFound(t *testing.T) {
	api := setupTagAPI(t)

	resp := api.Put("/api/tags/nonexistent", map[string]any{
		"slug": "x",
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestPatchTagSlug(t *testing.T) {
	api := setupTagAPI(t)

	createResp := api.Post("/api/tags", map[string]any{"slug": "original"})
	var created handlers.TagBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Patch("/api/tags/"+created.PublicID, map[string]any{
		"slug": "patched",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.TagBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.Slug != "patched" {
		t.Errorf("expected slug %q, got %q", "patched", body.Slug)
	}
}

func TestPatchTagNotFound(t *testing.T) {
	api := setupTagAPI(t)

	resp := api.Patch("/api/tags/nonexistent", map[string]any{
		"slug": "nope",
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestDeleteTag(t *testing.T) {
	api := setupTagAPI(t)

	createResp := api.Post("/api/tags", map[string]any{"slug": "bye"})
	var created handlers.TagBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Delete("/api/tags/" + created.PublicID)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, resp.Code, resp.Body.String())
	}

	// Verify it's gone
	getResp := api.Get("/api/tags/" + created.PublicID)
	if getResp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d after delete, got %d", http.StatusNotFound, getResp.Code)
	}
}

func TestDeleteTagNotFound(t *testing.T) {
	api := setupTagAPI(t)

	resp := api.Delete("/api/tags/nonexistent")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}
