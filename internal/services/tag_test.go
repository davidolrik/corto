package services_test

import (
	"context"
	"testing"

	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/davidolrik/corto/internal/services"
	"github.com/google/uuid"
)

func TestTagColorAndDescriptionRoundTrip(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)

	svc := services.NewTagService(testLogger(), db)
	created, err := svc.CreateTag(ctx, &handlers.TagData{
		Slug:        "tag-" + uuid.NewString(),
		Color:       "#ff6600",
		Description: "Campaign links",
	})
	if err != nil {
		t.Fatalf("creating tag: %v", err)
	}
	t.Cleanup(func() {
		_, err := db.NewDelete().Model((*model.Tag)(nil)).Where("public_id = ?", created.PublicID).Exec(ctx0)
		if err != nil {
			t.Errorf("cleaning up tag: %v", err)
		}
	})

	got, err := svc.GetTag(ctx, created.PublicID)
	if err != nil {
		t.Fatalf("getting tag: %v", err)
	}
	if got.Color != "#ff6600" {
		t.Errorf("expected color %q, got %q", "#ff6600", got.Color)
	}
	if got.Description != "Campaign links" {
		t.Errorf("expected description %q, got %q", "Campaign links", got.Description)
	}

	// Patching just the color keeps the description
	newColor := "#00aa66"
	patched, err := svc.PatchTag(ctx, created.PublicID, &handlers.TagPatch{Color: &newColor})
	if err != nil {
		t.Fatalf("patching tag: %v", err)
	}
	if patched.Color != "#00aa66" {
		t.Errorf("expected patched color %q, got %q", "#00aa66", patched.Color)
	}
	if patched.Description != "Campaign links" {
		t.Errorf("expected description to survive patch, got %q", patched.Description)
	}
}
