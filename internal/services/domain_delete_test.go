package services_test

import (
	"context"
	"testing"

	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/davidolrik/corto/internal/services"
	"github.com/google/uuid"
)

// TestDeleteDomainWithVisitsAndOrphans proves a domain with visit history can
// be deleted, that links also living on other domains survive, and that links
// existing only on the deleted domain are removed instead of orphaned.
func TestDeleteDomainWithVisitsAndOrphans(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domainA := createTestDomain(t, db, ctx)
	domainB := createTestDomain(t, db, ctx)

	shortCodeService := services.NewShortCodeService(testLogger(), db)
	onlyA, err := shortCodeService.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      "only-a-" + uuid.NewString(),
		TargetURL: "https://example.com/a",
		Domains:   []string{domainA.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code: %v", err)
	}
	both, err := shortCodeService.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      "both-" + uuid.NewString(),
		TargetURL: "https://example.com/both",
		Domains:   []string{domainA.FQDN, domainB.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, both.PublicID) })

	// Visits on domain A for both links
	var shortURLs []model.ShortURL
	err = db.NewSelect().Model(&shortURLs).
		Join("JOIN domains AS d ON d.id = su.domain_id").
		Where("d.fqdn = ?", domainA.FQDN).
		Scan(ctx0)
	if err != nil {
		t.Fatalf("loading short urls: %v", err)
	}
	for _, su := range shortURLs {
		visit := &model.Visit{PublicID: uuid.NewString(), ShortURLID: su.ID}
		if _, err := db.NewInsert().Model(visit).Exec(ctx0); err != nil {
			t.Fatalf("inserting visit: %v", err)
		}
	}

	// Deleting domain A must succeed despite the visit history
	domainService := services.NewDomainService(testLogger(), db)
	if err := domainService.DeleteDomain(ctx, domainA.PublicID); err != nil {
		t.Fatalf("deleting domain with visits: %v", err)
	}

	// The link that only lived on domain A is gone entirely
	orphans, err := db.NewSelect().Model((*model.ShortCode)(nil)).
		Where("public_id = ?", onlyA.PublicID).
		Count(ctx0)
	if err != nil {
		t.Fatalf("counting orphans: %v", err)
	}
	if orphans != 0 {
		t.Errorf("expected the single-domain link to be deleted with its domain")
	}

	// The link on both domains survives on domain B
	kept, err := shortCodeService.GetShortCode(ctx, both.PublicID)
	if err != nil {
		t.Fatalf("loading surviving link: %v", err)
	}
	if len(kept.Domains) != 1 || kept.Domains[0] != domainB.FQDN {
		t.Errorf("expected surviving link on %s only, got %v", domainB.FQDN, kept.Domains)
	}
}
