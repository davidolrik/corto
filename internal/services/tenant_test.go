package services_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"aidanwoods.dev/go-paseto"
	"github.com/davidolrik/corto/internal/auth"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/services"
	"github.com/google/uuid"
)

// slugifyName mirrors the service's slug derivation for assertions.
func slugifyName(name string) string {
	return strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "-"), "-")
}

func TestCreateTenant(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()

	owner := createTestUser(t, db, "owner password")

	svc := services.NewTenantService(testLogger(), db)
	tenant, err := svc.CreateTenant(ctx, "Tenant "+uuid.NewString(), "", owner.Username)
	if err != nil {
		t.Fatalf("creating tenant: %v", err)
	}
	t.Cleanup(func() { deleteTestTenant(t, db, tenant.ID) })

	if tenant.PublicID == "" {
		t.Error("expected public ID to be set")
	}
	if tenant.Slug == "" || tenant.Slug != slugifyName(tenant.Name) {
		t.Errorf("expected slug derived from name, got %q for %q", tenant.Slug, tenant.Name)
	}
	if tenant.OwnerID != owner.ID {
		t.Errorf("expected owner ID %d, got %d", owner.ID, tenant.OwnerID)
	}

	// The owner must get admin access to the new tenant
	access := &model.TenantUserAccess{}
	err = db.NewSelect().Model(access).
		Where("tenant_id = ?", tenant.ID).
		Where("user_id = ?", owner.ID).
		Scan(ctx)
	if err != nil {
		t.Fatalf("expected access row for owner: %v", err)
	}
	if !access.IsAdmin {
		t.Error("expected owner access to be admin")
	}
}

func TestCreateTenantUnknownOwner(t *testing.T) {
	db := testDatabase(t)

	svc := services.NewTenantService(testLogger(), db)
	_, err := svc.CreateTenant(context.Background(), "Orphan Tenant", "orphan-"+uuid.NewString(), "no-such-user-"+uuid.NewString())
	if err == nil {
		t.Fatal("expected error for unknown owner")
	}
}

func TestCreateTenantValidation(t *testing.T) {
	db := testDatabase(t)

	owner := createTestUser(t, db, "owner password")

	svc := services.NewTenantService(testLogger(), db)
	if _, err := svc.CreateTenant(context.Background(), "", "", owner.Username); err == nil {
		t.Error("expected error for empty tenant name")
	}
}

// TestBootstrapLogin proves the full bootstrap flow: create user, create
// tenant, then authenticate against it.
func TestBootstrapLogin(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()

	user := createTestUser(t, db, "bootstrap password")

	tenantService := services.NewTenantService(testLogger(), db)
	tenant, err := tenantService.CreateTenant(ctx, "Tenant "+uuid.NewString(), "", user.Username)
	if err != nil {
		t.Fatalf("creating tenant: %v", err)
	}
	t.Cleanup(func() { deleteTestTenant(t, db, tenant.ID) })

	secretKey := paseto.NewV4AsymmetricSecretKey()
	authService, err := services.NewAuthService(testLogger(), db, secretKey.ExportHex())
	if err != nil {
		t.Fatalf("creating auth service: %v", err)
	}

	result, err := authService.Login(ctx, user.Username, "bootstrap password", tenant.Slug)
	if err != nil {
		t.Fatalf("logging in: %v", err)
	}
	if result.Token == "" {
		t.Error("expected a token")
	}
	if !result.IsAdmin {
		t.Error("expected owner to be admin")
	}
	if result.TenantPublicID != tenant.PublicID {
		t.Errorf("expected tenant public ID %q, got %q", tenant.PublicID, result.TenantPublicID)
	}
	if result.TenantName != tenant.Name {
		t.Errorf("expected tenant name %q, got %q", tenant.Name, result.TenantName)
	}
	if result.TenantSlug != tenant.Slug {
		t.Errorf("expected tenant slug %q, got %q", tenant.Slug, result.TenantSlug)
	}
	if len(result.Tenants) != 1 {
		t.Errorf("expected 1 tenant membership, got %d", len(result.Tenants))
	}
}

// TestMultiTenantLoginAndSwitch proves a user with two tenants can log in
// without naming one, sees both memberships, and can switch.
func TestMultiTenantLoginAndSwitch(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()

	user := createTestUser(t, db, "multi password")
	first := createTestTenant(t, db, user)
	second := createTestTenant(t, db, user)

	secretKey := paseto.NewV4AsymmetricSecretKey()
	authService, err := services.NewAuthService(testLogger(), db, secretKey.ExportHex())
	if err != nil {
		t.Fatalf("creating auth service: %v", err)
	}

	// Login without a tenant slug picks one and lists both memberships
	result, err := authService.Login(ctx, user.Username, "multi password", "")
	if err != nil {
		t.Fatalf("logging in: %v", err)
	}
	if len(result.Tenants) != 2 {
		t.Fatalf("expected 2 tenant memberships, got %d", len(result.Tenants))
	}

	// Switch to the tenant that is not currently active
	target := first
	if result.TenantSlug == first.Slug {
		target = second
	}
	switchCtx := auth.WithClaims(ctx, auth.Claims{UserPublicID: user.PublicID, TenantPublicID: result.TenantPublicID})
	switched, err := authService.SwitchTenant(switchCtx, target.Slug)
	if err != nil {
		t.Fatalf("switching tenant: %v", err)
	}
	if switched.TenantSlug != target.Slug {
		t.Errorf("expected active tenant %q, got %q", target.Slug, switched.TenantSlug)
	}
	if switched.Token == "" {
		t.Error("expected a fresh token for the switched tenant")
	}

	// Switching to a tenant the user is not a member of fails
	otherUser := createTestUser(t, db, "other password")
	otherTenant := createTestTenant(t, db, otherUser)
	if _, err := authService.SwitchTenant(switchCtx, otherTenant.Slug); err == nil {
		t.Error("expected switching to a foreign tenant to fail")
	}
}
