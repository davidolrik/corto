package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/davidolrik/corto/internal/server/handlers"
)

// memoryShortCodeStore is an in-memory implementation of ShortCodeStore for testing.
type memoryShortCodeStore struct {
	mu    sync.RWMutex
	codes map[string]*handlers.ShortCodeData
	seq   int
}

func newMemoryShortCodeStore() *memoryShortCodeStore {
	return &memoryShortCodeStore{
		codes: make(map[string]*handlers.ShortCodeData),
	}
}

// slugConflicts reports whether another short code already uses the slug on
// any of the given domains, mirroring the database's unique constraint.
func (s *memoryShortCodeStore) slugConflicts(publicID, slug string, domains []string) bool {
	for _, other := range s.codes {
		if other.PublicID == publicID || other.Slug != slug {
			continue
		}
		for _, existing := range other.Domains {
			for _, domain := range domains {
				if existing == domain {
					return true
				}
			}
		}
	}
	return false
}

func (s *memoryShortCodeStore) CreateShortCode(_ context.Context, sc *handlers.ShortCodeData) (*handlers.ShortCodeData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sc.Slug == "" {
		// Mirrors the real store's slug generation
		sc.Slug = fmt.Sprintf("gen%d", s.seq+1)
	}
	if s.slugConflicts("", sc.Slug, sc.Domains) {
		return nil, fmt.Errorf("slug %q is already in use: %w", sc.Slug, handlers.ErrConflict)
	}
	s.seq++
	sc.PublicID = fmt.Sprintf("sc_%d", s.seq)
	now := time.Now().Truncate(time.Second)
	sc.CreatedAt = now
	sc.UpdatedAt = now
	s.codes[sc.PublicID] = sc
	return sc, nil
}

func (s *memoryShortCodeStore) GetShortCode(_ context.Context, publicID string) (*handlers.ShortCodeData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.codes[publicID]
	if !ok {
		return nil, fmt.Errorf("short code %q %w", publicID, handlers.ErrNotFound)
	}
	return sc, nil
}

func (s *memoryShortCodeStore) ListShortCodes(_ context.Context) ([]*handlers.ShortCodeData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*handlers.ShortCodeData, 0, len(s.codes))
	for _, sc := range s.codes {
		result = append(result, sc)
	}
	return result, nil
}

func (s *memoryShortCodeStore) UpdateShortCode(_ context.Context, publicID string, sc *handlers.ShortCodeData) (*handlers.ShortCodeData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.codes[publicID]
	if !ok {
		return nil, fmt.Errorf("short code %q %w", publicID, handlers.ErrNotFound)
	}
	if s.slugConflicts(publicID, sc.Slug, sc.Domains) {
		return nil, fmt.Errorf("slug %q is already in use: %w", sc.Slug, handlers.ErrConflict)
	}
	sc.PublicID = publicID
	sc.CreatedAt = existing.CreatedAt
	sc.UpdatedAt = time.Now().Truncate(time.Second)
	s.codes[publicID] = sc
	return sc, nil
}

func (s *memoryShortCodeStore) PatchShortCode(_ context.Context, publicID string, patch *handlers.ShortCodePatch) (*handlers.ShortCodeData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.codes[publicID]
	if !ok {
		return nil, fmt.Errorf("short code %q %w", publicID, handlers.ErrNotFound)
	}
	if patch.Title != nil {
		existing.Title = *patch.Title
	}
	if patch.Description != nil {
		existing.Description = *patch.Description
	}
	if patch.Slug != nil {
		existing.Slug = *patch.Slug
	}
	if patch.TargetURL != nil {
		existing.TargetURL = *patch.TargetURL
	}
	if patch.FallbackURL != nil {
		existing.FallbackURL = *patch.FallbackURL
	}
	if patch.IsCrawlable != nil {
		existing.IsCrawlable = *patch.IsCrawlable
	}
	if patch.ForwardQuery != nil {
		existing.ForwardQuery = *patch.ForwardQuery
	}
	if patch.ValidSince != nil {
		existing.ValidSince = patch.ValidSince
	}
	if patch.ValidUntil != nil {
		existing.ValidUntil = patch.ValidUntil
	}
	if patch.Domains != nil {
		existing.Domains = *patch.Domains
	}
	if patch.Tags != nil {
		existing.Tags = *patch.Tags
	}
	existing.UpdatedAt = time.Now().Truncate(time.Second)
	return existing, nil
}

func (s *memoryShortCodeStore) DeleteShortCode(_ context.Context, publicID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.codes[publicID]; !ok {
		return fmt.Errorf("short code %q %w", publicID, handlers.ErrNotFound)
	}
	delete(s.codes, publicID)
	return nil
}

func setupShortCodeAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	store := newMemoryShortCodeStore()
	handlers.RegisterShortCodeRoutes(api, store)
	return api
}

func TestCreateShortCode(t *testing.T) {
	api := setupShortCodeAPI(t)

	resp := api.Post("/api/short-codes", map[string]any{
		"slug":       "go",
		"target_url": "https://go.dev",
		"title":      "Go Website",
		"domains":    []string{"short.io", "s.example.com"},
		"tags":       []string{"programming", "golang"},
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var body handlers.ShortCodeBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if body.PublicID == "" {
		t.Fatal("expected public_id to be set")
	}
	if body.Slug != "go" {
		t.Errorf("expected slug %q, got %q", "go", body.Slug)
	}
	if body.TargetURL != "https://go.dev" {
		t.Errorf("expected target_url %q, got %q", "https://go.dev", body.TargetURL)
	}
	if body.Title != "Go Website" {
		t.Errorf("expected title %q, got %q", "Go Website", body.Title)
	}
	if !slices.Equal(body.Domains, []string{"short.io", "s.example.com"}) {
		t.Errorf("expected domains %v, got %v", []string{"short.io", "s.example.com"}, body.Domains)
	}
	if !slices.Equal(body.Tags, []string{"programming", "golang"}) {
		t.Errorf("expected tags %v, got %v", []string{"programming", "golang"}, body.Tags)
	}
	if body.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestCreateShortCodeWithoutOptionalFields(t *testing.T) {
	api := setupShortCodeAPI(t)

	resp := api.Post("/api/short-codes", map[string]any{
		"slug":       "minimal",
		"target_url": "https://example.com",
		"domains":    []string{"short.io"},
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var body handlers.ShortCodeBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.Title != "" {
		t.Errorf("expected empty title, got %q", body.Title)
	}
	if len(body.Tags) != 0 {
		t.Errorf("expected empty tags, got %v", body.Tags)
	}
	if len(body.Domains) != 1 || body.Domains[0] != "short.io" {
		t.Errorf("expected domains [short.io], got %v", body.Domains)
	}
}

func TestShortCodeDescriptionRoundTrip(t *testing.T) {
	api := setupShortCodeAPI(t)

	resp := api.Post("/api/short-codes", map[string]any{
		"slug":        "described",
		"target_url":  "https://example.com",
		"description": "Landing page for the spring campaign",
		"domains":     []string{"a.io"},
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var created handlers.ShortCodeBody
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding create response: %v", err)
	}
	if created.Description != "Landing page for the spring campaign" {
		t.Errorf("expected description %q, got %q", "Landing page for the spring campaign", created.Description)
	}

	patchResp := api.Patch("/api/short-codes/"+created.PublicID, map[string]any{
		"description": "Updated description",
	})
	if patchResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, patchResp.Code, patchResp.Body.String())
	}

	var patched handlers.ShortCodeBody
	if err := json.Unmarshal(patchResp.Body.Bytes(), &patched); err != nil {
		t.Fatalf("decoding patch response: %v", err)
	}
	if patched.Description != "Updated description" {
		t.Errorf("expected patched description %q, got %q", "Updated description", patched.Description)
	}
	if patched.Slug != "described" {
		t.Errorf("expected slug to be unchanged, got %q", patched.Slug)
	}
}

func TestCreateShortCodeWithoutSlug(t *testing.T) {
	api := setupShortCodeAPI(t)

	// The slug is optional on create; the store generates one
	resp := api.Post("/api/short-codes", map[string]any{
		"target_url": "https://example.com",
		"domains":    []string{"short.io"},
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var body handlers.ShortCodeBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Slug == "" {
		t.Error("expected a generated slug in the response")
	}
}

func TestCreateShortCodeMissingRequired(t *testing.T) {
	api := setupShortCodeAPI(t)

	resp := api.Post("/api/short-codes", map[string]any{
		"title": "Missing fields",
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}
}

func TestCreateShortCodeDuplicateSlugOnDomain(t *testing.T) {
	api := setupShortCodeAPI(t)

	resp := api.Post("/api/short-codes", map[string]any{
		"slug":       "dup",
		"target_url": "https://example.com/first",
		"domains":    []string{"a.io"},
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	// Same slug on the same domain conflicts
	resp = api.Post("/api/short-codes", map[string]any{
		"slug":       "dup",
		"target_url": "https://example.com/second",
		"domains":    []string{"a.io"},
	})
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, resp.Code, resp.Body.String())
	}

	// Same slug on another domain is fine
	resp = api.Post("/api/short-codes", map[string]any{
		"slug":       "dup",
		"target_url": "https://example.com/third",
		"domains":    []string{"b.io"},
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}
}

func TestUpdateShortCodeSlugConflict(t *testing.T) {
	api := setupShortCodeAPI(t)

	api.Post("/api/short-codes", map[string]any{
		"slug":       "one",
		"target_url": "https://example.com/one",
		"domains":    []string{"a.io"},
	})
	createResp := api.Post("/api/short-codes", map[string]any{
		"slug":       "two",
		"target_url": "https://example.com/two",
		"domains":    []string{"a.io"},
	})
	var created struct {
		PublicID string `json:"public_id"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding create response: %v", err)
	}

	// Renaming "two" to "one" on the same domain conflicts
	resp := api.Put("/api/short-codes/"+created.PublicID, map[string]any{
		"slug":       "one",
		"target_url": "https://example.com/two",
		"domains":    []string{"a.io"},
	})
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, resp.Code, resp.Body.String())
	}
}

func TestCreateShortCodeEmptyDomains(t *testing.T) {
	api := setupShortCodeAPI(t)

	// A short code without domains is unreachable and must be rejected
	resp := api.Post("/api/short-codes", map[string]any{
		"slug":       "orphan",
		"target_url": "https://example.com",
		"domains":    []string{},
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}
}

func TestUpdateShortCodeEmptyDomains(t *testing.T) {
	api := setupShortCodeAPI(t)

	createResp := api.Post("/api/short-codes", map[string]any{
		"slug":       "keep",
		"target_url": "https://example.com",
		"domains":    []string{"a.io"},
	})
	var created struct {
		PublicID string `json:"public_id"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding create response: %v", err)
	}

	resp := api.Put("/api/short-codes/"+created.PublicID, map[string]any{
		"slug":       "keep",
		"target_url": "https://example.com",
		"domains":    []string{},
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}
}

func TestCreateShortCodeMultipleDomains(t *testing.T) {
	api := setupShortCodeAPI(t)

	domains := []string{"a.io", "b.io", "c.io"}
	resp := api.Post("/api/short-codes", map[string]any{
		"slug":       "multi",
		"target_url": "https://example.com",
		"domains":    domains,
	})

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var body handlers.ShortCodeBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if !slices.Equal(body.Domains, domains) {
		t.Errorf("expected domains %v, got %v", domains, body.Domains)
	}
}

func TestGetShortCode(t *testing.T) {
	api := setupShortCodeAPI(t)

	createResp := api.Post("/api/short-codes", map[string]any{
		"slug":       "go",
		"target_url": "https://go.dev",
		"domains":    []string{"short.io"},
		"tags":       []string{"lang"},
	})
	var created handlers.ShortCodeBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Get("/api/short-codes/" + created.PublicID)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.ShortCodeBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.PublicID != created.PublicID {
		t.Errorf("expected public_id %q, got %q", created.PublicID, body.PublicID)
	}
	if body.Slug != "go" {
		t.Errorf("expected slug %q, got %q", "go", body.Slug)
	}
	if !slices.Equal(body.Domains, []string{"short.io"}) {
		t.Errorf("expected domains %v, got %v", []string{"short.io"}, body.Domains)
	}
	if !slices.Equal(body.Tags, []string{"lang"}) {
		t.Errorf("expected tags %v, got %v", []string{"lang"}, body.Tags)
	}
}

func TestGetShortCodeNotFound(t *testing.T) {
	api := setupShortCodeAPI(t)

	resp := api.Get("/api/short-codes/nonexistent")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestListShortCodes(t *testing.T) {
	api := setupShortCodeAPI(t)

	// Empty list
	resp := api.Get("/api/short-codes")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	var empty []handlers.ShortCodeBody
	json.Unmarshal(resp.Body.Bytes(), &empty)
	if len(empty) != 0 {
		t.Errorf("expected 0 short codes, got %d", len(empty))
	}

	// Create two
	api.Post("/api/short-codes", map[string]any{
		"slug": "one", "target_url": "https://one.com", "domains": []string{"s.io"},
	})
	api.Post("/api/short-codes", map[string]any{
		"slug": "two", "target_url": "https://two.com", "domains": []string{"s.io"},
	})

	resp = api.Get("/api/short-codes")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	var codes []handlers.ShortCodeBody
	json.Unmarshal(resp.Body.Bytes(), &codes)
	if len(codes) != 2 {
		t.Errorf("expected 2 short codes, got %d", len(codes))
	}
}

func TestUpdateShortCode(t *testing.T) {
	api := setupShortCodeAPI(t)

	createResp := api.Post("/api/short-codes", map[string]any{
		"slug":       "old",
		"target_url": "https://old.com",
		"domains":    []string{"short.io"},
		"tags":       []string{"old-tag"},
	})
	var created handlers.ShortCodeBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Put("/api/short-codes/"+created.PublicID, map[string]any{
		"slug":       "replaced",
		"target_url": "https://replaced.com",
		"title":      "Replaced",
		"domains":    []string{"new.io", "other.io"},
		"tags":       []string{"new-tag"},
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.ShortCodeBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.PublicID != created.PublicID {
		t.Errorf("expected public_id %q, got %q", created.PublicID, body.PublicID)
	}
	if body.Slug != "replaced" {
		t.Errorf("expected slug %q, got %q", "replaced", body.Slug)
	}
	if body.TargetURL != "https://replaced.com" {
		t.Errorf("expected target_url %q, got %q", "https://replaced.com", body.TargetURL)
	}
	if !slices.Equal(body.Domains, []string{"new.io", "other.io"}) {
		t.Errorf("expected domains %v, got %v", []string{"new.io", "other.io"}, body.Domains)
	}
	if !slices.Equal(body.Tags, []string{"new-tag"}) {
		t.Errorf("expected tags %v, got %v", []string{"new-tag"}, body.Tags)
	}
	if body.CreatedAt != created.CreatedAt {
		t.Error("expected created_at to be preserved")
	}
}

func TestUpdateShortCodeNotFound(t *testing.T) {
	api := setupShortCodeAPI(t)

	resp := api.Put("/api/short-codes/nonexistent", map[string]any{
		"slug":       "x",
		"target_url": "https://x.com",
		"domains":    []string{"s.io"},
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestPatchShortCodeSingleField(t *testing.T) {
	api := setupShortCodeAPI(t)

	createResp := api.Post("/api/short-codes", map[string]any{
		"slug":       "original",
		"target_url": "https://original.com",
		"title":      "Original Title",
		"domains":    []string{"short.io"},
		"tags":       []string{"tag1"},
	})
	var created handlers.ShortCodeBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	// Patch only the title
	resp := api.Patch("/api/short-codes/"+created.PublicID, map[string]any{
		"title": "Patched Title",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.ShortCodeBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.Title != "Patched Title" {
		t.Errorf("expected title %q, got %q", "Patched Title", body.Title)
	}
	// Other fields should be unchanged
	if body.Slug != "original" {
		t.Errorf("expected slug %q to be unchanged, got %q", "original", body.Slug)
	}
	if body.TargetURL != "https://original.com" {
		t.Errorf("expected target_url %q to be unchanged, got %q", "https://original.com", body.TargetURL)
	}
	if !slices.Equal(body.Domains, []string{"short.io"}) {
		t.Errorf("expected domains %v to be unchanged, got %v", []string{"short.io"}, body.Domains)
	}
	if !slices.Equal(body.Tags, []string{"tag1"}) {
		t.Errorf("expected tags %v to be unchanged, got %v", []string{"tag1"}, body.Tags)
	}
}

func TestPatchShortCodeDomains(t *testing.T) {
	api := setupShortCodeAPI(t)

	createResp := api.Post("/api/short-codes", map[string]any{
		"slug":       "test",
		"target_url": "https://test.com",
		"domains":    []string{"a.io"},
	})
	var created handlers.ShortCodeBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	// Patch only domains
	resp := api.Patch("/api/short-codes/"+created.PublicID, map[string]any{
		"domains": []string{"a.io", "b.io", "c.io"},
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.ShortCodeBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if !slices.Equal(body.Domains, []string{"a.io", "b.io", "c.io"}) {
		t.Errorf("expected domains %v, got %v", []string{"a.io", "b.io", "c.io"}, body.Domains)
	}
	// Slug should be unchanged
	if body.Slug != "test" {
		t.Errorf("expected slug %q to be unchanged, got %q", "test", body.Slug)
	}
}

func TestPatchShortCodeTags(t *testing.T) {
	api := setupShortCodeAPI(t)

	createResp := api.Post("/api/short-codes", map[string]any{
		"slug":       "test",
		"target_url": "https://test.com",
		"domains":    []string{"s.io"},
		"tags":       []string{"old"},
	})
	var created handlers.ShortCodeBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	// Patch only tags
	resp := api.Patch("/api/short-codes/"+created.PublicID, map[string]any{
		"tags": []string{"new1", "new2"},
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.ShortCodeBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if !slices.Equal(body.Tags, []string{"new1", "new2"}) {
		t.Errorf("expected tags %v, got %v", []string{"new1", "new2"}, body.Tags)
	}
	// Domains should be unchanged
	if !slices.Equal(body.Domains, []string{"s.io"}) {
		t.Errorf("expected domains %v to be unchanged, got %v", []string{"s.io"}, body.Domains)
	}
}

func TestPatchShortCodeNotFound(t *testing.T) {
	api := setupShortCodeAPI(t)

	resp := api.Patch("/api/short-codes/nonexistent", map[string]any{
		"title": "nope",
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestDeleteShortCode(t *testing.T) {
	api := setupShortCodeAPI(t)

	createResp := api.Post("/api/short-codes", map[string]any{
		"slug":       "bye",
		"target_url": "https://bye.com",
		"domains":    []string{"short.io"},
	})
	var created handlers.ShortCodeBody
	json.Unmarshal(createResp.Body.Bytes(), &created)

	resp := api.Delete("/api/short-codes/" + created.PublicID)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, resp.Code, resp.Body.String())
	}

	// Verify it's gone
	getResp := api.Get("/api/short-codes/" + created.PublicID)
	if getResp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d after delete, got %d", http.StatusNotFound, getResp.Code)
	}
}

func TestDeleteShortCodeNotFound(t *testing.T) {
	api := setupShortCodeAPI(t)

	resp := api.Delete("/api/short-codes/nonexistent")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}
