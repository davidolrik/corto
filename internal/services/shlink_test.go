package services_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
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
	mux.HandleFunc("GET /rest/v3/domains", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			http.Error(w, `{"title":"Invalid API key"}`, http.StatusUnauthorized)
			return
		}
		fmt.Fprintf(w, `{
			"domains": {
				"data": [
					{"domain": "%s", "isDefault": true},
					{"domain": "%s", "isDefault": false}
				]
			}
		}`, defaultDomainFQDN, customDomainFQDN)
	})
	mux.HandleFunc("GET /rest/v3/short-urls", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			http.Error(w, `{"title":"Invalid API key"}`, http.StatusUnauthorized)
			return
		}
		page := r.URL.Query().Get("page")
		if page == "2" {
			// promo also exists on the custom domain with the same target,
			// which corto represents as one link on two domains
			fmt.Fprintf(w, `{
				"shortUrls": {
					"data": [
						{
							"shortCode": "promo",
							"longUrl": "https://example.com/landing",
							"domain": "%s",
							"title": "Spring promo",
							"tags": ["spring"],
							"crawlable": true,
							"forwardQuery": true,
							"meta": {"validSince": null, "validUntil": null}
						},
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
			}`, customDomainFQDN)
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
		// The custom domain's promo has its own visit history
		if r.URL.Query().Get("domain") == customDomainFQDN {
			fmt.Fprint(w, `{
				"visits": {
					"data": [
						{
							"referer": "",
							"date": "2026-05-03T12:00:00+00:00",
							"userAgent": "imported-agent/3.0",
							"visitLocation": null
						}
					],
					"pagination": {"currentPage": 1, "pagesCount": 1}
				}
			}`)
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
	if summary.Merged != 1 {
		t.Errorf("expected 1 merged domain, got %d", summary.Merged)
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
	if summary.Visits != 3 {
		t.Errorf("expected 3 imported visits, got %d", summary.Visits)
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

	// Same code on two Shlink domains becomes one corto link on both domains
	if len(promo.Domains) != 2 {
		t.Fatalf("expected promo on 2 domains, got %v", promo.Domains)
	}
	if promo.Visits != 3 {
		t.Errorf("expected 3 visits on promo, got %d", promo.Visits)
	}
	if promo.VisitsByDomain[defaultDomain] != 2 || promo.VisitsByDomain[customDomain] != 1 {
		t.Errorf("expected per domain visits 2 and 1, got %v", promo.VisitsByDomain)
	}
	if promo.VisitsByCountry["DK"] != 1 || promo.VisitsByCountry["unknown"] != 2 {
		t.Errorf("expected countries DK 1 and unknown 2, got %v", promo.VisitsByCountry)
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

// TestImportShlinkIsIdempotent proves re-imports change nothing, and that
// the visit history can be imported in a later run.
func TestImportShlinkIsIdempotent(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)

	defaultDomain := "imp-" + uuid.NewString() + ".example.com"
	customDomain := "custom-" + uuid.NewString() + ".example.com"
	shlink := fakeShlink(t, defaultDomain, customDomain)

	shortCodeService := services.NewShortCodeService(testLogger(), db)
	t.Cleanup(func() {
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

	importer := services.NewShlinkImporter(testLogger(), db)
	opts := services.ShlinkImportOptions{
		BaseURL:       shlink.URL,
		APIKey:        "test-key",
		TenantSlug:    tenant.Slug,
		DefaultDomain: defaultDomain,
	}

	// First run without visits
	first, err := importer.Import(ctx0, opts)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if first.ShortCodes != 3 || first.Merged != 1 || first.Visits != 0 {
		t.Errorf("first run: expected 3 links, 1 merged, 0 visits, got %+v", first)
	}

	// Second run brings the visit history without duplicating links
	opts.WithVisits = true
	second, err := importer.Import(ctx0, opts)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.ShortCodes != 0 || second.Merged != 0 || second.Domains != 0 || second.Tags != 0 {
		t.Errorf("second run: expected no link changes, got %+v", second)
	}
	if second.Visits != 3 {
		t.Errorf("second run: expected 3 imported visits, got %d", second.Visits)
	}
	if second.Unchanged != 4 {
		t.Errorf("second run: expected 4 unchanged entries, got %d", second.Unchanged)
	}

	// Third run is a zero import
	third, err := importer.Import(ctx0, opts)
	if err != nil {
		t.Fatalf("third import: %v", err)
	}
	if third.ShortCodes != 0 || third.Merged != 0 || third.Visits != 0 || third.Domains != 0 || third.Tags != 0 {
		t.Errorf("third run: expected a zero import, got %+v", third)
	}

	// The visit history exists exactly once
	codes, err := shortCodeService.ListShortCodes(ctx)
	if err != nil {
		t.Fatalf("listing short codes: %v", err)
	}
	for _, sc := range codes {
		if sc.Slug == "promo" && sc.Visits != 3 {
			t.Errorf("expected 3 visits on promo, got %d", sc.Visits)
		}
	}
}

// TestImportShlinkDetectsDefaultDomain proves links on Shlink's default
// domain land on that domain's real name when --domain is not given.
func TestImportShlinkDetectsDefaultDomain(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)

	defaultDomain := "imp-" + uuid.NewString() + ".example.com"
	customDomain := "custom-" + uuid.NewString() + ".example.com"
	shlink := fakeShlink(t, defaultDomain, customDomain)

	shortCodeService := services.NewShortCodeService(testLogger(), db)
	t.Cleanup(func() {
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

	// No DefaultDomain given; the importer asks Shlink
	importer := services.NewShlinkImporter(testLogger(), db)
	summary, err := importer.Import(ctx0, services.ShlinkImportOptions{
		BaseURL:    shlink.URL,
		APIKey:     "test-key",
		TenantSlug: tenant.Slug,
	})
	if err != nil {
		t.Fatalf("importing without --domain: %v", err)
	}
	if summary.ShortCodes != 3 {
		t.Errorf("expected 3 imported links, got %d", summary.ShortCodes)
	}
	if summary.DefaultDomain != defaultDomain {
		t.Errorf("expected the summary to report default domain %q, got %q", defaultDomain, summary.DefaultDomain)
	}

	codes, err := shortCodeService.ListShortCodes(ctx)
	if err != nil {
		t.Fatalf("listing short codes: %v", err)
	}
	for _, sc := range codes {
		if sc.Slug == "promo" {
			if !slices.Contains(sc.Domains, defaultDomain) {
				t.Errorf("expected promo on Shlink's default domain %s, got %v", defaultDomain, sc.Domains)
			}
		}
	}
}

// TestImportShlinkSkipsMergeConflicts proves that extending an imported link
// to a domain where a different link already holds the slug skips the entry
// instead of aborting.
func TestImportShlinkSkipsMergeConflicts(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)

	defaultDomain := "imp-" + uuid.NewString() + ".example.com"
	customDomain := "custom-" + uuid.NewString() + ".example.com"
	shlink := fakeShlink(t, defaultDomain, customDomain)

	// A different link already holds "promo" on the custom domain, so the
	// merge of the imported promo onto that domain must be skipped
	domainService := services.NewDomainService(testLogger(), db)
	if _, err := domainService.CreateDomain(ctx, &handlers.DomainData{FQDN: customDomain}); err != nil {
		t.Fatalf("creating custom domain: %v", err)
	}
	shortCodeService := services.NewShortCodeService(testLogger(), db)
	if _, err := shortCodeService.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:      "promo",
		TargetURL: "https://example.com/a-different-promo",
		Domains:   []string{customDomain},
	}); err != nil {
		t.Fatalf("creating conflicting short code: %v", err)
	}

	t.Cleanup(func() {
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

	importer := services.NewShlinkImporter(testLogger(), db)
	summary, err := importer.Import(ctx0, services.ShlinkImportOptions{
		BaseURL:       shlink.URL,
		APIKey:        "test-key",
		TenantSlug:    tenant.Slug,
		DefaultDomain: defaultDomain,
		WithVisits:    true,
	})
	if err != nil {
		t.Fatalf("import must not abort on a merge conflict: %v", err)
	}

	// promo (default), docs, dup import; the custom-domain promo is skipped
	if summary.ShortCodes != 3 {
		t.Errorf("expected 3 imported links, got %d", summary.ShortCodes)
	}
	if summary.Skipped != 1 {
		t.Errorf("expected 1 skipped entry, got %d", summary.Skipped)
	}

	// The imported promo stays on the default domain only, and the skipped
	// entry's visits must not land on the conflicting link
	codes, err := shortCodeService.ListShortCodes(ctx)
	if err != nil {
		t.Fatalf("listing short codes: %v", err)
	}
	for _, sc := range codes {
		if sc.Slug != "promo" {
			continue
		}
		switch sc.TargetURL {
		case "https://example.com/landing":
			if len(sc.Domains) != 1 || sc.Domains[0] != defaultDomain {
				t.Errorf("expected imported promo on %s only, got %v", defaultDomain, sc.Domains)
			}
			if sc.Visits != 2 {
				t.Errorf("expected 2 visits on imported promo, got %d", sc.Visits)
			}
		case "https://example.com/a-different-promo":
			if sc.Visits != 0 {
				t.Errorf("expected no visits on the conflicting promo, got %d", sc.Visits)
			}
		}
	}
}

// TestImportShlinkSkipsForeignDomains proves a domain owned by another
// tenant skips its links without aborting the import.
func TestImportShlinkSkipsForeignDomains(t *testing.T) {
	db := testDatabase(t)
	ctx0 := context.Background()

	user := createTestUser(t, db, "password")
	tenant := createTestTenant(t, db, user)
	ctx := claimsContext(user, tenant)

	otherUser := createTestUser(t, db, "password")
	otherTenant := createTestTenant(t, db, otherUser)

	defaultDomain := "imp-" + uuid.NewString() + ".example.com"
	customDomain := "custom-" + uuid.NewString() + ".example.com"
	shlink := fakeShlink(t, defaultDomain, customDomain)

	// The custom domain already belongs to the other tenant
	otherDomainService := services.NewDomainService(testLogger(), db)
	foreign, err := otherDomainService.CreateDomain(claimsContext(otherUser, otherTenant), &handlers.DomainData{FQDN: customDomain})
	if err != nil {
		t.Fatalf("creating foreign domain: %v", err)
	}

	shortCodeService := services.NewShortCodeService(testLogger(), db)
	t.Cleanup(func() {
		codes, err := shortCodeService.ListShortCodes(ctx)
		if err != nil {
			t.Errorf("listing short codes for cleanup: %v", err)
			return
		}
		for _, sc := range codes {
			deleteTestShortCode(t, db, sc.PublicID)
		}
		for _, table := range []any{(*model.Domain)(nil), (*model.Tag)(nil)} {
			for _, tenantID := range []int{tenant.ID, otherTenant.ID} {
				if _, err := db.NewDelete().Model(table).Where("tenant_id = ?", tenantID).Exec(ctx0); err != nil {
					t.Errorf("cleaning up tenant data: %v", err)
				}
			}
		}
	})
	_ = foreign

	importer := services.NewShlinkImporter(testLogger(), db)
	summary, err := importer.Import(ctx0, services.ShlinkImportOptions{
		BaseURL:       shlink.URL,
		APIKey:        "test-key",
		TenantSlug:    tenant.Slug,
		DefaultDomain: defaultDomain,
	})
	if err != nil {
		t.Fatalf("import must not abort on a foreign domain: %v", err)
	}

	// docs and the custom-domain promo entry are skipped; promo and dup import
	if summary.ShortCodes != 2 {
		t.Errorf("expected 2 imported links, got %d", summary.ShortCodes)
	}
	if summary.Skipped != 2 {
		t.Errorf("expected 2 skipped entries, got %d", summary.Skipped)
	}
}
