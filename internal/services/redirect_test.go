package services_test

import (
	"context"
	"testing"

	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/davidolrik/corto/internal/services"
	"github.com/google/uuid"
)

// fakeCountryResolver resolves IPs from a fixed map.
type fakeCountryResolver map[string]string

func (f fakeCountryResolver) Country(ip string) string {
	return f[ip]
}

func TestResolveRedirectFillsVisitLimit(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domain := createTestDomain(t, db, ctx)

	maxVisits := 2
	shortCodeService := services.NewShortCodeService(testLogger(), db)
	created, err := shortCodeService.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      "slug-" + uuid.NewString(),
		TargetURL: "https://example.com/landing",
		MaxVisits: &maxVisits,
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, created.PublicID) })

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

	svc := services.NewRedirectService(testLogger(), db, nil)
	target, err := svc.ResolveRedirect(ctx0, domain.FQDN, created.Slug)
	if err != nil {
		t.Fatalf("resolving redirect: %v", err)
	}
	if target.ShortURL == nil {
		t.Fatal("expected the slug to resolve")
	}
	if target.ShortURL.MaxVisits == nil || *target.ShortURL.MaxVisits != 2 {
		t.Errorf("expected max visits 2, got %v", target.ShortURL.MaxVisits)
	}
	if target.ShortURL.Visits != 1 {
		t.Errorf("expected 1 recorded visit, got %d", target.ShortURL.Visits)
	}
}

func TestRecordVisitResolvesCountry(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domain := createTestDomain(t, db, ctx)

	shortCodeService := services.NewShortCodeService(testLogger(), db)
	created, err := shortCodeService.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      "slug-" + uuid.NewString(),
		TargetURL: "https://example.com/landing",
		Domains:   []string{domain.FQDN},
	})
	if err != nil {
		t.Fatalf("creating short code: %v", err)
	}
	t.Cleanup(func() { deleteTestShortCode(t, db, created.PublicID) })

	shortURL := &model.ShortURL{}
	err = db.NewSelect().Model(shortURL).
		Where("su.short_code_id IN (?)", db.NewSelect().Model((*model.ShortCode)(nil)).
			Column("id").Where("public_id = ?", created.PublicID)).
		Scan(ctx0)
	if err != nil {
		t.Fatalf("loading short url: %v", err)
	}

	resolver := fakeCountryResolver{"81.2.69.142": "GB"}
	svc := services.NewRedirectService(testLogger(), db, resolver)

	err = svc.RecordVisit(ctx0, &handlers.VisitData{
		ShortURLPublicID: shortURL.PublicID,
		IPAddress:        "81.2.69.142",
		UserAgent:        "test-agent/1.0",
	})
	if err != nil {
		t.Fatalf("recording visit: %v", err)
	}

	visit := &model.Visit{}
	err = db.NewSelect().Model(visit).Where("short_url_id = ?", shortURL.ID).Scan(ctx0)
	if err != nil {
		t.Fatalf("loading visit: %v", err)
	}
	if visit.Country != "GB" {
		t.Errorf("expected country %q, got %q", "GB", visit.Country)
	}

	// A nil resolver records visits without a country
	withoutResolver := services.NewRedirectService(testLogger(), db, nil)
	err = withoutResolver.RecordVisit(ctx0, &handlers.VisitData{
		ShortURLPublicID: shortURL.PublicID,
		IPAddress:        "81.2.69.142",
	})
	if err != nil {
		t.Fatalf("recording visit without resolver: %v", err)
	}
}
