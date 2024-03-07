package services_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/davidolrik/corto/internal/auth"
	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/davidolrik/corto/internal/services"
	"github.com/google/uuid"
)

func claimsContext(user *model.User, tenant *model.Tenant) context.Context {
	return auth.WithClaims(context.Background(), auth.Claims{
		UserPublicID:   user.PublicID,
		TenantPublicID: tenant.PublicID,
		IsAdmin:        true,
	})
}

func createTestDomain(t *testing.T, db core.Database, ctx context.Context) *handlers.DomainData {
	t.Helper()

	svc := services.NewDomainService(testLogger(), db)
	domain, err := svc.CreateDomain(ctx, &handlers.DomainData{
		FQDN: "d-" + uuid.NewString() + ".example.com",
	})
	if err != nil {
		t.Fatalf("creating test domain: %v", err)
	}
	t.Cleanup(func() {
		_, err := db.NewDelete().Model((*model.Domain)(nil)).
			Where("public_id = ?", domain.PublicID).
			Exec(context.Background())
		if err != nil {
			t.Errorf("cleaning up domain %s: %v", domain.PublicID, err)
		}
	})
	return domain
}

func deleteTestShortCode(t *testing.T, db core.Database, publicID string) {
	t.Helper()

	ctx := context.Background()
	ids := db.NewSelect().Model((*model.ShortCode)(nil)).
		Column("id").
		Where("public_id = ?", publicID)
	shortURLIDs := db.NewSelect().Model((*model.ShortURL)(nil)).
		Column("id").
		Where("short_code_id IN (?)", ids)
	if _, err := db.NewDelete().Model((*model.Visit)(nil)).Where("short_url_id IN (?)", shortURLIDs).Exec(ctx); err != nil {
		t.Errorf("cleaning up visits for %s: %v", publicID, err)
	}
	if _, err := db.NewDelete().Model((*model.ShortURL)(nil)).Where("short_code_id IN (?)", ids).Exec(ctx); err != nil {
		t.Errorf("cleaning up short urls for %s: %v", publicID, err)
	}
	if _, err := db.NewDelete().Model((*model.ShortCodeTag)(nil)).Where("shortcode_id IN (?)", ids).Exec(ctx); err != nil {
		t.Errorf("cleaning up short code tags for %s: %v", publicID, err)
	}
	if _, err := db.NewDelete().Model((*model.ShortCode)(nil)).Where("public_id = ?", publicID).Exec(ctx); err != nil {
		t.Errorf("cleaning up short code %s: %v", publicID, err)
	}
}

func TestUpdateShortCodeWithVisitsPreservesHistory(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domain := createTestDomain(t, db, ctx)

	svc := services.NewShortCodeService(testLogger(), db)
	slug := "slug-" + uuid.NewString()
	created, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      slug,
		TargetURL: "https://example.com/landing",
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, created.PublicID) })

	// Record a visit on the short URL, as the redirect handler would
	shortURL := &model.ShortURL{}
	err = db.NewSelect().Model(shortURL).
		Where("su.short_code_id IN (?)", db.NewSelect().Model((*model.ShortCode)(nil)).
			Column("id").Where("public_id = ?", created.PublicID)).
		Scan(ctx0)
	if err != nil {
		t.Fatalf("loading short url: %v", err)
	}
	visit := &model.Visit{PublicID: uuid.NewString(), ShortURLID: shortURL.ID}
	if _, err := db.NewInsert().Model(visit).Exec(ctx0); err != nil {
		t.Fatalf("inserting visit: %v", err)
	}

	// Editing the short code must neither fail nor lose the visit history
	_, err = svc.UpdateShortCode(ctx, created.PublicID, &handlers.ShortCodeData{
		Slug:      slug,
		Title:     "Edited title",
		TargetURL: "https://example.com/landing",
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("updating short code with visits: %v", err)
	}

	kept := &model.ShortURL{}
	err = db.NewSelect().Model(kept).Where("su.id = ?", shortURL.ID).Scan(ctx0)
	if err != nil {
		t.Fatalf("expected short url row to survive the edit: %v", err)
	}
	if kept.PublicID != shortURL.PublicID {
		t.Errorf("expected short url public ID %q to be preserved, got %q", shortURL.PublicID, kept.PublicID)
	}

	visits, err := db.NewSelect().Model((*model.Visit)(nil)).
		Where("short_url_id = ?", shortURL.ID).
		Count(ctx0)
	if err != nil {
		t.Fatalf("counting visits: %v", err)
	}
	if visits != 1 {
		t.Errorf("expected 1 visit to survive the edit, got %d", visits)
	}
}

func TestCreateShortCodeGeneratesSlug(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domain := createTestDomain(t, db, ctx)

	svc := services.NewShortCodeService(testLogger(), db)

	first, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		TargetURL: "https://example.com/first",
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code without slug: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, first.PublicID) })

	if first.Slug == "" {
		t.Fatal("expected a generated slug")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9]{7}$`).MatchString(first.Slug) {
		t.Errorf("expected 7 character base62 slug, got %q", first.Slug)
	}

	// The generated slug resolves like any other
	shortURL := &model.ShortURL{}
	err = db.NewSelect().Model(shortURL).
		Where("su.short_code_id IN (?)", db.NewSelect().Model((*model.ShortCode)(nil)).
			Column("id").Where("public_id = ?", first.PublicID)).
		Scan(ctx0)
	if err != nil {
		t.Fatalf("loading short url: %v", err)
	}
	if shortURL.Slug != first.Slug {
		t.Errorf("expected short url slug %q, got %q", first.Slug, shortURL.Slug)
	}

	// A second generated link on the same domain gets a distinct slug
	second, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		TargetURL: "https://example.com/second",
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("creating second short code without slug: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, second.PublicID) })

	if second.Slug == first.Slug {
		t.Errorf("expected distinct generated slugs, both are %q", first.Slug)
	}
}

func TestShortCodeSlugUniquePerDomain(t *testing.T) {
	db := testDatabase(t)

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domainA := createTestDomain(t, db, ctx)
	domainB := createTestDomain(t, db, ctx)

	svc := services.NewShortCodeService(testLogger(), db)
	slug := "slug-" + uuid.NewString()

	first, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      slug,
		TargetURL: "https://example.com/first",
		Domains:   []string{domainA.FQDN},
	})
	if err != nil {
		t.Fatalf("creating first short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, first.PublicID) })

	// Same slug on the same domain must conflict
	duplicate, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      slug,
		TargetURL: "https://example.com/second",
		Domains:   []string{domainA.FQDN},
	})
	if duplicate != nil {
		t.Cleanup(func() { deleteTestShortCode(t, db, duplicate.PublicID) })
	}
	if !errors.Is(err, handlers.ErrConflict) {
		t.Fatalf("expected ErrConflict for duplicate slug on same domain, got: %v", err)
	}

	// Same slug on another domain is fine
	other, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      slug,
		TargetURL: "https://example.com/other",
		Domains:   []string{domainB.FQDN},
	})
	if err != nil {
		t.Fatalf("expected same slug on another domain to be allowed, got: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, other.PublicID) })
}

func TestShortCodeSlugRenameFreesSlug(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domain := createTestDomain(t, db, ctx)

	svc := services.NewShortCodeService(testLogger(), db)
	oldSlug := "slug-" + uuid.NewString()
	newSlug := "slug-" + uuid.NewString()

	created, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      oldSlug,
		TargetURL: "https://example.com/landing",
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, created.PublicID) })

	_, err = svc.UpdateShortCode(ctx, created.PublicID, &handlers.ShortCodeData{
		Slug:      newSlug,
		TargetURL: "https://example.com/landing",
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("renaming slug: %v", err)
	}

	// The denormalized slug on short_urls must follow the rename
	shortURL := &model.ShortURL{}
	err = db.NewSelect().Model(shortURL).
		Where("su.short_code_id IN (?)", db.NewSelect().Model((*model.ShortCode)(nil)).
			Column("id").Where("public_id = ?", created.PublicID)).
		Scan(ctx0)
	if err != nil {
		t.Fatalf("loading short url: %v", err)
	}
	if shortURL.Slug != newSlug {
		t.Errorf("expected short url slug %q after rename, got %q", newSlug, shortURL.Slug)
	}

	// The old slug is free for a new short code on the same domain
	reuse, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      oldSlug,
		TargetURL: "https://example.com/reuse",
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("expected renamed-away slug to be reusable, got: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, reuse.PublicID) })
}

func TestUpdateShortCodeRollsBackOnFailure(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domain := createTestDomain(t, db, ctx)

	svc := services.NewShortCodeService(testLogger(), db)
	slug := "slug-" + uuid.NewString()
	created, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      slug,
		TargetURL: "https://example.com/landing",
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, created.PublicID) })

	// An update that fails partway (unknown tag) must leave no trace of the
	// earlier steps: neither the slug change nor the domain sync may persist.
	_, err = svc.UpdateShortCode(ctx, created.PublicID, &handlers.ShortCodeData{
		Slug:      slug + "-renamed",
		TargetURL: "https://example.com/landing",
		Domains:   []string{domain.FQDN},
		Tags:      []string{"no-such-tag-" + uuid.NewString()},
	})
	if err == nil {
		t.Fatal("expected update with unknown tag to fail")
	}

	shortCode := &model.ShortCode{}
	err = db.NewSelect().Model(shortCode).Where("sc.public_id = ?", created.PublicID).Scan(ctx0)
	if err != nil {
		t.Fatalf("loading short code: %v", err)
	}
	if shortCode.Slug != slug {
		t.Errorf("expected slug %q after failed update, got %q", slug, shortCode.Slug)
	}

	shortURL := &model.ShortURL{}
	err = db.NewSelect().Model(shortURL).Where("su.short_code_id = ?", shortCode.ID).Scan(ctx0)
	if err != nil {
		t.Fatalf("loading short url: %v", err)
	}
	if shortURL.Slug != slug {
		t.Errorf("expected short url slug %q after failed update, got %q", slug, shortURL.Slug)
	}
}

func TestShortCodeVisitCounts(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domainA := createTestDomain(t, db, ctx)
	domainB := createTestDomain(t, db, ctx)

	svc := services.NewShortCodeService(testLogger(), db)
	visited, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      "slug-" + uuid.NewString(),
		TargetURL: "https://example.com/landing",
		Domains:   []string{domainA.FQDN, domainB.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, visited.PublicID) })

	unvisited, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      "slug-" + uuid.NewString(),
		TargetURL: "https://example.com/other",
		Domains:   []string{domainA.FQDN},
	})
	if err != nil {
		t.Fatalf("creating second short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, unvisited.PublicID) })

	// Record two recent "spring" campaign visits from GB via domain A and
	// one ten-day-old visit without campaign or country via domain B
	recordVisits := func(fqdn string, count int, at time.Time, campaign, country string) {
		t.Helper()
		shortURL := &model.ShortURL{}
		err := db.NewSelect().Model(shortURL).
			Join("JOIN domains AS d ON d.id = su.domain_id").
			Where("d.fqdn = ?", fqdn).
			Where("su.short_code_id IN (?)", db.NewSelect().Model((*model.ShortCode)(nil)).
				Column("id").Where("public_id = ?", visited.PublicID)).
			Scan(ctx0)
		if err != nil {
			t.Fatalf("loading short url for %s: %v", fqdn, err)
		}
		for range count {
			visit := &model.Visit{
				PublicID:   uuid.NewString(),
				ShortURLID: shortURL.ID,
				Campaign:   campaign,
				Country:    country,
				CreatedAt:  at,
				UpdatedAt:  at,
			}
			if _, err := db.NewInsert().Model(visit).Exec(ctx0); err != nil {
				t.Fatalf("inserting visit: %v", err)
			}
		}
	}
	recordVisits(domainA.FQDN, 2, time.Now(), "spring", "GB")
	recordVisits(domainB.FQDN, 1, time.Now().AddDate(0, 0, -10), "", "")

	byPublicID := map[string]*handlers.ShortCodeData{}
	shortCodes, err := svc.ListShortCodes(ctx)
	if err != nil {
		t.Fatalf("listing short codes: %v", err)
	}
	for _, sc := range shortCodes {
		byPublicID[sc.PublicID] = sc
	}
	if got := byPublicID[visited.PublicID].Visits; got != 3 {
		t.Errorf("expected 3 visits on visited link, got %d", got)
	}
	if got := byPublicID[visited.PublicID].VisitsThisWeek; got != 2 {
		t.Errorf("expected 2 visits this week on visited link, got %d", got)
	}
	if got := byPublicID[visited.PublicID].VisitsByDomain[domainA.FQDN]; got != 2 {
		t.Errorf("expected 2 visits via %s, got %d", domainA.FQDN, got)
	}
	if got := byPublicID[visited.PublicID].VisitsByDomain[domainB.FQDN]; got != 1 {
		t.Errorf("expected 1 visit via %s, got %d", domainB.FQDN, got)
	}
	if got := byPublicID[visited.PublicID].VisitsByCampaign["spring"]; got != 2 {
		t.Errorf("expected 2 visits for campaign spring, got %d", got)
	}
	if got := byPublicID[visited.PublicID].VisitsByCampaign["direct"]; got != 1 {
		t.Errorf("expected 1 direct visit, got %d", got)
	}
	if got := byPublicID[visited.PublicID].VisitsByCountry["GB"]; got != 2 {
		t.Errorf("expected 2 visits from GB, got %d", got)
	}
	if got := byPublicID[visited.PublicID].VisitsByCountry["unknown"]; got != 1 {
		t.Errorf("expected 1 visit with unknown country, got %d", got)
	}
	if got := byPublicID[unvisited.PublicID].Visits; got != 0 {
		t.Errorf("expected 0 visits on unvisited link, got %d", got)
	}

	got, err := svc.GetShortCode(ctx, visited.PublicID)
	if err != nil {
		t.Fatalf("getting short code: %v", err)
	}
	if got.Visits != 3 {
		t.Errorf("expected 3 visits from GetShortCode, got %d", got.Visits)
	}
	if got.VisitsByDomain[domainA.FQDN] != 2 {
		t.Errorf("expected 2 visits via %s from GetShortCode, got %d", domainA.FQDN, got.VisitsByDomain[domainA.FQDN])
	}
	if got.VisitsByCampaign["spring"] != 2 {
		t.Errorf("expected 2 spring visits from GetShortCode, got %d", got.VisitsByCampaign["spring"])
	}
}

func TestListShortCodesEmptyTenant(t *testing.T) {
	db := testDatabase(t)

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)

	svc := services.NewShortCodeService(testLogger(), db)
	shortCodes, err := svc.ListShortCodes(ctx)
	if err != nil {
		t.Fatalf("listing short codes for empty tenant: %v", err)
	}
	if len(shortCodes) != 0 {
		t.Errorf("expected no short codes, got %d", len(shortCodes))
	}
}

func TestShortCodeLifecycleScopedToTenant(t *testing.T) {
	db := testDatabase(t)

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domain := createTestDomain(t, db, ctx)

	svc := services.NewShortCodeService(testLogger(), db)

	slug := "slug-" + uuid.NewString()
	created, err := svc.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:        slug,
		Description: "Landing page link",
		TargetURL:   "https://example.com/landing",
		Domains:     []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, created.PublicID) })

	// The owning tenant sees the short code
	shortCodes, err := svc.ListShortCodes(ctx)
	if err != nil {
		t.Fatalf("listing short codes: %v", err)
	}
	if len(shortCodes) != 1 {
		t.Fatalf("expected 1 short code, got %d", len(shortCodes))
	}
	if shortCodes[0].Slug != slug {
		t.Errorf("expected slug %q, got %q", slug, shortCodes[0].Slug)
	}
	if len(shortCodes[0].Domains) != 1 || shortCodes[0].Domains[0] != domain.FQDN {
		t.Errorf("expected domains [%s], got %v", domain.FQDN, shortCodes[0].Domains)
	}

	got, err := svc.GetShortCode(ctx, created.PublicID)
	if err != nil {
		t.Fatalf("getting short code: %v", err)
	}
	if got.Slug != slug {
		t.Errorf("expected slug %q, got %q", slug, got.Slug)
	}
	if got.Description != "Landing page link" {
		t.Errorf("expected description %q, got %q", "Landing page link", got.Description)
	}

	// Another tenant must not see it
	otherUser := createTestUser(t, db, "password")
	otherTenant := createTestTenant(t, db, otherUser)
	otherCtx := claimsContext(otherUser, otherTenant)

	otherShortCodes, err := svc.ListShortCodes(otherCtx)
	if err != nil {
		t.Fatalf("listing short codes for other tenant: %v", err)
	}
	if len(otherShortCodes) != 0 {
		t.Errorf("expected other tenant to see no short codes, got %d", len(otherShortCodes))
	}
	if _, err := svc.GetShortCode(otherCtx, created.PublicID); err == nil {
		t.Error("expected other tenant to not find the short code")
	}
}
