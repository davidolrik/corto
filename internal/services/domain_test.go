package services_test

import (
	"testing"

	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/davidolrik/corto/internal/services"
)

func TestDomainDescriptionRoundTrip(t *testing.T) {
	db := testDatabase(t)

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)

	domain := createTestDomain(t, db, ctx)

	svc := services.NewDomainService(testLogger(), db)
	description := "Primary short link domain"
	patched, err := svc.PatchDomain(ctx, domain.PublicID, &handlers.DomainPatch{Description: &description})
	if err != nil {
		t.Fatalf("patching domain: %v", err)
	}
	if patched.Description != description {
		t.Errorf("expected description %q, got %q", description, patched.Description)
	}
	if patched.FQDN != domain.FQDN {
		t.Errorf("expected fqdn to be unchanged, got %q", patched.FQDN)
	}

	got, err := svc.GetDomain(ctx, domain.PublicID)
	if err != nil {
		t.Fatalf("getting domain: %v", err)
	}
	if got.Description != description {
		t.Errorf("expected description %q from GetDomain, got %q", description, got.Description)
	}
}
