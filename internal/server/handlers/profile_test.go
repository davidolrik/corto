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

type fakeProfileStore struct {
	currentPassword string
	newPassword     string
}

func (s *fakeProfileStore) GetProfile(_ context.Context) (*handlers.ProfileData, error) {
	return &handlers.ProfileData{PublicID: "user_1", Username: "alice"}, nil
}

func (s *fakeProfileStore) ChangePassword(_ context.Context, currentPassword, newPassword string) error {
	if currentPassword != s.currentPassword {
		return fmt.Errorf("wrong password: %w", handlers.ErrInvalidCredentials)
	}
	s.newPassword = newPassword
	return nil
}

func setupProfileAPI(t *testing.T) (humatest.TestAPI, *fakeProfileStore) {
	t.Helper()
	_, api := humatest.New(t)
	store := &fakeProfileStore{currentPassword: "old-password"}
	handlers.RegisterProfileRoutes(api, store)
	return api, store
}

func TestGetProfile(t *testing.T) {
	api, _ := setupProfileAPI(t)

	resp := api.Get("/api/profile")

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Username != "alice" {
		t.Errorf("expected username %q, got %q", "alice", body.Username)
	}
}

func TestChangePassword(t *testing.T) {
	api, store := setupProfileAPI(t)

	resp := api.Put("/api/profile/password", map[string]any{
		"current_password": "old-password",
		"new_password":     "brand-new-password",
	})

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, resp.Code, resp.Body.String())
	}
	if store.newPassword != "brand-new-password" {
		t.Errorf("expected new password to be stored, got %q", store.newPassword)
	}
}

func TestChangePasswordWrongCurrent(t *testing.T) {
	api, _ := setupProfileAPI(t)

	resp := api.Put("/api/profile/password", map[string]any{
		"current_password": "not-the-password",
		"new_password":     "brand-new-password",
	})

	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, resp.Code, resp.Body.String())
	}
}

func TestChangePasswordTooShort(t *testing.T) {
	api, _ := setupProfileAPI(t)

	resp := api.Put("/api/profile/password", map[string]any{
		"current_password": "old-password",
		"new_password":     "short",
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}
}
