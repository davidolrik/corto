package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"aidanwoods.dev/go-paseto"
	"github.com/davidolrik/corto/internal/auth"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/davidolrik/corto/internal/services"
	"golang.org/x/crypto/bcrypt"
)

func TestCreateUser(t *testing.T) {
	db := testDatabase(t)

	user := createTestUser(t, db, "correct horse battery staple")

	if user.ID == 0 {
		t.Error("expected user ID to be set")
	}
	if user.PublicID == "" {
		t.Error("expected public ID to be set")
	}
	if user.Password == "correct horse battery staple" {
		t.Error("expected password to be hashed, got plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("correct horse battery staple")); err != nil {
		t.Errorf("expected stored hash to verify against the password: %v", err)
	}
}

func TestCreateUserDuplicateUsername(t *testing.T) {
	db := testDatabase(t)
	svc := services.NewUserService(testLogger(), db)

	user := createTestUser(t, db, "first password")

	_, err := svc.CreateUser(context.Background(), user.Username, "second password")
	if err == nil {
		t.Fatal("expected error for duplicate username")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "first password")
	svc := services.NewUserService(testLogger(), db)
	ctx := auth.WithClaims(ctx0, auth.Claims{UserPublicID: user.PublicID})

	// The wrong current password is rejected
	err := svc.ChangePassword(ctx, "not the password", "replacement password")
	if !errors.Is(err, handlers.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}

	// The right current password changes it
	if err := svc.ChangePassword(ctx, "first password", "replacement password"); err != nil {
		t.Fatalf("changing password: %v", err)
	}

	profile, err := svc.GetProfile(ctx)
	if err != nil {
		t.Fatalf("getting profile: %v", err)
	}
	if profile.Username != user.Username {
		t.Errorf("expected username %q, got %q", user.Username, profile.Username)
	}

	// The new password works for login, the old one no longer does
	secretKey := paseto.NewV4AsymmetricSecretKey()
	authService, err := services.NewAuthService(testLogger(), db, secretKey.ExportHex())
	if err != nil {
		t.Fatalf("creating auth service: %v", err)
	}
	createTestTenant(t, db, user)
	if _, err := authService.Login(ctx0, user.Username, "replacement password", ""); err != nil {
		t.Errorf("expected login with new password to succeed: %v", err)
	}
	if _, err := authService.Login(ctx0, user.Username, "first password", ""); err == nil {
		t.Error("expected login with old password to fail")
	}
}

func TestCreateUserValidation(t *testing.T) {
	db := testDatabase(t)
	svc := services.NewUserService(testLogger(), db)

	if _, err := svc.CreateUser(context.Background(), "", "password"); err == nil {
		t.Error("expected error for empty username")
	}
	if _, err := svc.CreateUser(context.Background(), "someuser", ""); err == nil {
		t.Error("expected error for empty password")
	}
}
