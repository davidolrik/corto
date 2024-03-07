package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/davidolrik/corto/internal/server/handlers"
)

type memoryAuthStore struct {
	// users maps username -> password
	users map[string]string
	// userPublicIDs maps username -> public_id
	userPublicIDs map[string]string
	// memberships maps username -> tenants in order
	memberships map[string][]handlers.TenantMembership
}

func newMemoryAuthStore() *memoryAuthStore {
	return &memoryAuthStore{
		users: map[string]string{
			"alice": "secret123",
			"bob":   "password456",
		},
		userPublicIDs: map[string]string{
			"alice": "user_1",
			"bob":   "user_2",
		},
		memberships: map[string][]handlers.TenantMembership{
			"alice": {
				{Slug: "tenant-one", Name: "Tenant One", IsAdmin: true},
				{Slug: "tenant-two", Name: "Tenant Two", IsAdmin: false},
			},
			"bob": {
				{Slug: "tenant-one", Name: "Tenant One", IsAdmin: false},
			},
		},
	}
}

func (s *memoryAuthStore) result(username, slug string) (*handlers.LoginResult, error) {
	all := s.memberships[username]
	if len(all) == 0 {
		return nil, fmt.Errorf("no tenant access")
	}
	active := all[0]
	if slug != "" {
		found := false
		for _, m := range all {
			if m.Slug == slug {
				active = m
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("no access to tenant %q", slug)
		}
	}
	return &handlers.LoginResult{
		Token:          "test-token-" + username + "-" + active.Slug,
		UserPublicID:   s.userPublicIDs[username],
		Username:       username,
		TenantPublicID: "pub-" + active.Slug,
		TenantSlug:     active.Slug,
		TenantName:     active.Name,
		IsAdmin:        active.IsAdmin,
		Tenants:        all,
	}, nil
}

func (s *memoryAuthStore) Login(_ context.Context, username, password, tenantSlug string) (*handlers.LoginResult, error) {
	storedPassword, ok := s.users[username]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	if storedPassword != password {
		return nil, fmt.Errorf("invalid password")
	}
	return s.result(username, tenantSlug)
}

func (s *memoryAuthStore) SwitchTenant(_ context.Context, tenantSlug string) (*handlers.LoginResult, error) {
	// The fake store acts as alice, mirroring claims-based resolution
	return s.result("alice", tenantSlug)
}

func setupAuthAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	store := newMemoryAuthStore()
	handlers.RegisterAuthRoutes(api, store)
	return api
}

func TestLoginSuccess(t *testing.T) {
	api := setupAuthAPI(t)

	resp := api.Post("/api/auth/login", map[string]any{
		"username": "alice",
		"password": "secret123",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.LoginBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if body.Token == "" {
		t.Fatal("expected token to be set")
	}
	if body.UserID != "user_1" {
		t.Errorf("expected user_id %q, got %q", "user_1", body.UserID)
	}
	if body.Username != "alice" {
		t.Errorf("expected username %q, got %q", "alice", body.Username)
	}
	if body.TenantSlug != "tenant-one" {
		t.Errorf("expected tenant_slug %q, got %q", "tenant-one", body.TenantSlug)
	}
	if body.TenantName != "Tenant One" {
		t.Errorf("expected tenant_name %q, got %q", "Tenant One", body.TenantName)
	}
	if !body.IsAdmin {
		t.Error("expected is_admin to be true for alice")
	}
	if len(body.Tenants) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(body.Tenants))
	}
	if body.Tenants[1].Slug != "tenant-two" {
		t.Errorf("expected second tenant slug %q, got %q", "tenant-two", body.Tenants[1].Slug)
	}
}

func TestLoginWithTenantSlug(t *testing.T) {
	api := setupAuthAPI(t)

	resp := api.Post("/api/auth/login", map[string]any{
		"username": "alice",
		"password": "secret123",
		"tenant":   "tenant-two",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.LoginBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.TenantSlug != "tenant-two" {
		t.Errorf("expected tenant_slug %q, got %q", "tenant-two", body.TenantSlug)
	}
	if body.IsAdmin {
		t.Error("expected is_admin to be false for alice on tenant-two")
	}
}

func TestSwitchTenant(t *testing.T) {
	api := setupAuthAPI(t)

	resp := api.Post("/api/auth/tenant", map[string]any{
		"tenant": "tenant-two",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.LoginBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.TenantSlug != "tenant-two" {
		t.Errorf("expected tenant_slug %q, got %q", "tenant-two", body.TenantSlug)
	}
	if body.Token == "" {
		t.Error("expected a fresh token")
	}
}

func TestSwitchTenantUnknown(t *testing.T) {
	api := setupAuthAPI(t)

	resp := api.Post("/api/auth/tenant", map[string]any{
		"tenant": "not-mine",
	})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestLoginNonAdmin(t *testing.T) {
	api := setupAuthAPI(t)

	resp := api.Post("/api/auth/login", map[string]any{
		"username": "bob",
		"password": "password456",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.LoginBody
	json.Unmarshal(resp.Body.Bytes(), &body)

	if body.IsAdmin {
		t.Error("expected is_admin to be false for bob")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	api := setupAuthAPI(t)

	resp := api.Post("/api/auth/login", map[string]any{
		"username": "alice",
		"password": "wrongpassword",
	})

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}
}

func TestLoginUnknownUser(t *testing.T) {
	api := setupAuthAPI(t)

	resp := api.Post("/api/auth/login", map[string]any{
		"username": "nobody",
		"password": "anything",
	})

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}
}

func TestLoginNoAccessToTenant(t *testing.T) {
	api := setupAuthAPI(t)

	resp := api.Post("/api/auth/login", map[string]any{
		"username": "alice",
		"password": "secret123",
		"tenant":   "tenant-99",
	})

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}
}

func TestLoginMissingRequired(t *testing.T) {
	api := setupAuthAPI(t)

	resp := api.Post("/api/auth/login", map[string]any{
		"username": "alice",
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}
}
