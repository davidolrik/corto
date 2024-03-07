package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/davidolrik/corto/internal/services"
	"github.com/google/uuid"
)

func TestGetStatsScopedToTenant(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)
	domain := createTestDomain(t, db, ctx)

	tagService := services.NewTagService(testLogger(), db)
	tag, err := tagService.CreateTag(ctx, &handlers.TagData{Slug: "tag-" + uuid.NewString()})
	if err != nil {
		t.Fatalf("creating tag: %v", err)
	}
	t.Cleanup(func() {
		_, err := db.NewDelete().Model((*model.Tag)(nil)).Where("public_id = ?", tag.PublicID).Exec(ctx0)
		if err != nil {
			t.Errorf("cleaning up tag: %v", err)
		}
	})

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

	// One recent visit from DK, one ten-day-old without a country
	shortURL := &model.ShortURL{}
	err = db.NewSelect().Model(shortURL).
		Where("su.short_code_id IN (?)", db.NewSelect().Model((*model.ShortCode)(nil)).
			Column("id").Where("public_id = ?", created.PublicID)).
		Scan(ctx0)
	if err != nil {
		t.Fatalf("loading short url: %v", err)
	}
	old := time.Now().AddDate(0, 0, -10)
	for _, visit := range []*model.Visit{
		{PublicID: uuid.NewString(), ShortURLID: shortURL.ID, Country: "DK", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{PublicID: uuid.NewString(), ShortURLID: shortURL.ID, CreatedAt: old, UpdatedAt: old},
	} {
		if _, err := db.NewInsert().Model(visit).Exec(ctx0); err != nil {
			t.Fatalf("inserting visit: %v", err)
		}
	}

	statsService := services.NewStatsService(testLogger(), db)
	stats, err := statsService.GetStats(ctx)
	if err != nil {
		t.Fatalf("getting stats: %v", err)
	}

	if stats.Links != 1 {
		t.Errorf("expected 1 link, got %d", stats.Links)
	}
	if stats.Domains != 1 {
		t.Errorf("expected 1 domain, got %d", stats.Domains)
	}
	if stats.Tags != 1 {
		t.Errorf("expected 1 tag, got %d", stats.Tags)
	}
	if stats.Visits != 2 {
		t.Errorf("expected 2 visits, got %d", stats.Visits)
	}
	if stats.VisitsThisWeek != 1 {
		t.Errorf("expected 1 visit this week, got %d", stats.VisitsThisWeek)
	}
	if stats.VisitsByCountry["DK"] != 1 {
		t.Errorf("expected 1 visit from DK, got %d", stats.VisitsByCountry["DK"])
	}
	if stats.VisitsByCountry["unknown"] != 1 {
		t.Errorf("expected 1 visit with unknown country, got %d", stats.VisitsByCountry["unknown"])
	}

	// Another tenant sees none of it
	otherUser := createTestUser(t, db, "password")
	otherTenant := createTestTenant(t, db, otherUser)
	otherStats, err := statsService.GetStats(claimsContext(otherUser, otherTenant))
	if err != nil {
		t.Fatalf("getting stats for other tenant: %v", err)
	}
	if otherStats.Links != 0 || otherStats.Visits != 0 || otherStats.Domains != 0 {
		t.Errorf("expected empty stats for other tenant, got %+v", otherStats)
	}
}
