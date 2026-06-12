package services_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/davidolrik/corto/internal/services"
	"github.com/google/uuid"
)

// fakeShlink serves a minimal Shlink REST API v3 with two pages of short
// URLs and a visit list for one of them.
func fakeShlink(t *testing.T, defaultDomainFQDN, customDomainFQDN string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /rest/v3/short-urls", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			http.Error(w, `{"title":"Invalid API key"}`, http.StatusUnauthorized)
			return
		}
		page := r.URL.Query().Get("page")
		if page == "2" {
			fmt.Fprintf(w, `{
				"shortUrls": {
					"data": [
						{
							"shortCode": "dup",
							"longUrl": "https://example.com/duplicate",
							"domain": null,
							"title": null,
							"tags": [],
							"crawlable": false,
							"forwardQuery": false,
							"meta": {"validSince": null, "validUntil": null}
						}
					],
					"pagination": {"currentPage": 2, "pagesCount": 2}
				}
			}`)
			return
		}
		fmt.Fprintf(w, `{
			"shortUrls": {
				"data": [
					{
						"shortCode": "promo",
						"longUrl": "https://example.com/landing",
						"domain": null,
						"title": "Spring promo",
						"tags": ["spring"],
						"crawlable": true,
						"forwardQuery": true,
						"meta": {"validSince": "2026-01-01T00:00:00+00:00", "validUntil": null, "maxVisits": 5}
					},
					{
						"shortCode": "docs",
						"longUrl": "https://example.com/docs",
						"domain": "%s",
						"title": null,
						"tags": [],
						"crawlable": false,
						"forwardQuery": false,
						"meta": {"validSince": null, "validUntil": null}
					}
				],
				"pagination": {"currentPage": 1, "pagesCount": 2}
			}
		}`, customDomainFQDN)
	})
	mux.HandleFunc("GET /rest/v3/short-urls/{code}/visits", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("code") != "promo" {
			fmt.Fprint(w, `{"visits": {"data": [], "pagination": {"currentPage": 1, "pagesCount": 1}}}`)
			return
		}
		fmt.Fprint(w, `{
			"visits": {
				"data": [
					{
						"referer": "https://social.example.com",
						"date": "2026-05-01T12:00:00+00:00",
						"userAgent": "imported-agent/1.0",
						"visitLocation": {"countryCode": "DK"}
					},
					{
						"referer": "",
						"date": "2026-05-02T12:00:00+00:00",
						"userAgent": "imported-agent/2.0",
						"visitLocation": null
					}
				],
				"pagination": {"currentPage": 1, "pagesCount": 1}
			}
		}`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestImportShlink(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)

	defaultDomain := "imp-" + uuid.NewString() + ".example.com"
	customDomain := "custom-" + uuid.NewString() + ".example.com"
	shlink := fakeShlink(t, defaultDomain, customDomain)

	// "dup" already exists on the default domain and must be skipped
	domainService := services.NewDomainService(testLogger(), db)
	if _, err := domainService.CreateDomain(ctx, &handlers.DomainData{FQDN: defaultDomain}); err != nil {
		t.Fatalf("creating default domain: %v", err)
	}
	shortCodeService := services.NewShortCodeService(testLogger(), db)
	preExisting, err := shortCodeService.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      "dup",
		TargetURL: "https://example.com/already-here",
		Domains:   []string{defaultDomain},
	})
	if err != nil {
		t.Fatalf("creating pre-existing short code: %v", err)
	}

	t.Cleanup(func() {
		// Imported short codes, then domains and tags of the test tenant
		codes, err := shortCodeService.ListShortCodes(ctx)
		if err != nil {
			t.Errorf("listing short codes for cleanup: %v", err)
			return
		}
		for _, sc := range codes {
			deleteTestShortCode(t, db, sc.PublicID)
		}
		for _, table := range []any{(*model.Domain)(nil), (*model.Tag)(nil)} {
			if _, err := db.NewDelete().Model(table).Where("tenant_id = ?", tenant.ID).Exec(ctx0); err != nil {
				t.Errorf("cleaning up tenant data: %v", err)
			}
		}
	})
	_ = preExisting

	importer := services.NewShlinkImporter(testLogger(), db)
	summary, err := importer.Import(ctx0, services.ShlinkImportOptions{
		BaseURL:       shlink.URL,
		APIKey:        "test-key",
		TenantSlug:    tenant.Slug,
		DefaultDomain: defaultDomain,
		WithVisits:    true,
	})
	if err != nil {
		t.Fatalf("importing from shlink: %v", err)
	}

	if summary.ShortCodes != 2 {
		t.Errorf("expected 2 imported short codes, got %d", summary.ShortCodes)
	}
	if summary.Skipped != 1 {
		t.Errorf("expected 1 skipped short code, got %d", summary.Skipped)
	}
	if summary.Domains != 1 {
		t.Errorf("expected 1 created domain (custom), got %d", summary.Domains)
	}
	if summary.Tags != 1 {
		t.Errorf("expected 1 created tag, got %d", summary.Tags)
	}
	if summary.Visits != 2 {
		t.Errorf("expected 2 imported visits, got %d", summary.Visits)
	}

	// The imported promo link carries everything over
	codes, err := shortCodeService.ListShortCodes(ctx)
	if err != nil {
		t.Fatalf("listing short codes: %v", err)
	}
	bySlug := map[string]*handlers.ShortCodeData{}
	for _, sc := range codes {
		bySlug[sc.Slug] = sc
	}

	promo := bySlug["promo"]
	if promo == nil {
		t.Fatal("expected promo to be imported")
	}
	if promo.Title != "Spring promo" {
		t.Errorf("expected title %q, got %q", "Spring promo", promo.Title)
	}
	if !promo.ForwardQuery || !promo.IsCrawlable {
		t.Errorf("expected forward_query and is_crawlable to carry over")
	}
	if promo.ValidSince == nil {
		t.Error("expected valid_since to carry over")
	}
	if promo.MaxVisits == nil || *promo.MaxVisits != 5 {
		t.Errorf("expected max_visits 5 to carry over, got %v", promo.MaxVisits)
	}
	if len(promo.Tags) != 1 || promo.Tags[0] != "spring" {
		t.Errorf("expected tags [spring], got %v", promo.Tags)
	}
	if promo.Visits != 2 {
		t.Errorf("expected 2 visits on promo, got %d", promo.Visits)
	}
	if promo.VisitsByCountry["DK"] != 1 || promo.VisitsByCountry["unknown"] != 1 {
		t.Errorf("expected countries DK and unknown, got %v", promo.VisitsByCountry)
	}

	docs := bySlug["docs"]
	if docs == nil {
		t.Fatal("expected docs to be imported")
	}
	if len(docs.Domains) != 1 || docs.Domains[0] != customDomain {
		t.Errorf("expected docs on %s, got %v", customDomain, docs.Domains)
	}

	// The pre-existing target of "dup" is untouched
	if bySlug["dup"].TargetURL != "https://example.com/already-here" {
		t.Errorf("expected dup to be skipped, got target %q", bySlug["dup"].TargetURL)
	}
}
