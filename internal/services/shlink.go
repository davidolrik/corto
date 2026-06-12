package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/davidolrik/corto/internal/auth"
	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/google/uuid"
)

// ShlinkImportOptions configures an import from a Shlink instance.
type ShlinkImportOptions struct {
	BaseURL       string // Shlink instance, e.g. https://s.example.com
	APIKey        string
	TenantSlug    string // corto tenant receiving the data
	DefaultDomain string // corto domain for Shlink's default domain
	WithVisits    bool   // also import the visit history
}

// ShlinkImportSummary counts what an import created.
type ShlinkImportSummary struct {
	Domains    int
	Tags       int
	ShortCodes int
	Skipped    int // short codes whose slug was already taken on the domain
	Visits     int
}

type ShlinkImporter struct {
	log    *slog.Logger
	db     core.Database
	client *http.Client
}

func NewShlinkImporter(log *slog.Logger, db core.Database) *ShlinkImporter {
	return &ShlinkImporter{
		log:    log,
		db:     db,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Shlink REST API v3 shapes, limited to the fields corto imports.
type shlinkShortURL struct {
	ShortCode    string   `json:"shortCode"`
	LongURL      string   `json:"longUrl"`
	Domain       *string  `json:"domain"`
	Title        *string  `json:"title"`
	Tags         []string `json:"tags"`
	Crawlable    bool     `json:"crawlable"`
	ForwardQuery bool     `json:"forwardQuery"`
	Meta         struct {
		ValidSince *time.Time `json:"validSince"`
		ValidUntil *time.Time `json:"validUntil"`
		MaxVisits  *int       `json:"maxVisits"`
	} `json:"meta"`
}

type shlinkPagination struct {
	CurrentPage int `json:"currentPage"`
	PagesCount  int `json:"pagesCount"`
}

type shlinkVisit struct {
	Referer       string    `json:"referer"`
	Date          time.Time `json:"date"`
	UserAgent     string    `json:"userAgent"`
	VisitLocation *struct {
		CountryCode string `json:"countryCode"`
	} `json:"visitLocation"`
}

func (s *ShlinkImporter) fetch(ctx context.Context, opts ShlinkImportOptions, path string, query url.Values, out any) error {
	requestURL := strings.TrimSuffix(opts.BaseURL, "/") + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("X-Api-Key", opts.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("calling shlink: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("shlink returned %s for %s", resp.Status, path)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding shlink response: %w", err)
	}
	return nil
}

// Import copies short URLs (and optionally their visits) from a Shlink
// instance into the given tenant. Domains and tags are created as needed;
// slugs already taken on their domain are skipped.
func (s *ShlinkImporter) Import(ctx context.Context, opts ShlinkImportOptions) (*ShlinkImportSummary, error) {
	tenant := &model.Tenant{}
	err := s.db.NewSelect().Model(tenant).Where("slug = ?", opts.TenantSlug).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant %q not found: %w", opts.TenantSlug, err)
	}

	// All writes go through the regular services, scoped to the tenant
	ctx = auth.WithClaims(ctx, auth.Claims{TenantPublicID: tenant.PublicID})
	domainService := NewDomainService(s.log, s.db)
	tagService := NewTagService(s.log, s.db)
	shortCodeService := NewShortCodeService(s.log, s.db)

	knownDomains := map[string]bool{}
	domains, err := domainService.ListDomains(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range domains {
		knownDomains[d.FQDN] = true
	}

	knownTags := map[string]bool{}
	tags, err := tagService.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tags {
		knownTags[t.Slug] = true
	}

	summary := &ShlinkImportSummary{}
	for page := 1; ; page++ {
		var response struct {
			ShortURLs struct {
				Data       []shlinkShortURL `json:"data"`
				Pagination shlinkPagination `json:"pagination"`
			} `json:"shortUrls"`
		}
		query := url.Values{"page": {fmt.Sprint(page)}, "itemsPerPage": {"100"}}
		if err := s.fetch(ctx, opts, "/rest/v3/short-urls", query, &response); err != nil {
			return nil, err
		}

		for _, item := range response.ShortURLs.Data {
			if err := s.importShortURL(ctx, opts, item, domainService, tagService, shortCodeService, knownDomains, knownTags, summary); err != nil {
				return nil, err
			}
		}

		if page >= response.ShortURLs.Pagination.PagesCount {
			break
		}
	}

	return summary, nil
}

func (s *ShlinkImporter) importShortURL(
	ctx context.Context,
	opts ShlinkImportOptions,
	item shlinkShortURL,
	domainService *DomainService,
	tagService *TagService,
	shortCodeService *ShortCodeService,
	knownDomains, knownTags map[string]bool,
	summary *ShlinkImportSummary,
) error {
	fqdn := opts.DefaultDomain
	if item.Domain != nil && *item.Domain != "" {
		fqdn = *item.Domain
	}
	if fqdn == "" {
		return fmt.Errorf("short URL %q uses Shlink's default domain; provide a corto domain with --domain", item.ShortCode)
	}

	if !knownDomains[fqdn] {
		_, err := domainService.CreateDomain(ctx, &handlers.DomainData{FQDN: fqdn, Description: "Imported from Shlink"})
		if err != nil {
			return fmt.Errorf("creating domain %q: %w", fqdn, err)
		}
		knownDomains[fqdn] = true
		summary.Domains++
	}

	for _, tag := range item.Tags {
		if knownTags[tag] {
			continue
		}
		if _, err := tagService.CreateTag(ctx, &handlers.TagData{Slug: tag}); err != nil {
			return fmt.Errorf("creating tag %q: %w", tag, err)
		}
		knownTags[tag] = true
		summary.Tags++
	}

	title := ""
	if item.Title != nil {
		title = *item.Title
	}
	_, err := shortCodeService.CreateShortCode(ctx, &handlers.ShortCodeData{
		Slug:         item.ShortCode,
		Title:        title,
		TargetURL:    item.LongURL,
		IsCrawlable:  item.Crawlable,
		ForwardQuery: item.ForwardQuery,
		ValidSince:   item.Meta.ValidSince,
		ValidUntil:   item.Meta.ValidUntil,
		MaxVisits:    item.Meta.MaxVisits,
		Domains:      []string{fqdn},
		Tags:         item.Tags,
	})
	if errors.Is(err, handlers.ErrConflict) {
		s.log.Info("Skipping short URL, slug already taken", "slug", item.ShortCode, "domain", fqdn)
		summary.Skipped++
		return nil
	}
	if err != nil {
		return fmt.Errorf("creating short code %q: %w", item.ShortCode, err)
	}
	summary.ShortCodes++

	if opts.WithVisits {
		return s.importVisits(ctx, opts, item, fqdn, summary)
	}
	return nil
}

func (s *ShlinkImporter) importVisits(ctx context.Context, opts ShlinkImportOptions, item shlinkShortURL, fqdn string, summary *ShlinkImportSummary) error {
	shortURL := &model.ShortURL{}
	err := s.db.NewSelect().Model(shortURL).
		Join("JOIN domains AS d ON d.id = su.domain_id").
		Where("su.slug = ?", item.ShortCode).
		Where("d.fqdn = ?", fqdn).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("loading imported short url %q: %w", item.ShortCode, err)
	}

	for page := 1; ; page++ {
		var response struct {
			Visits struct {
				Data       []shlinkVisit    `json:"data"`
				Pagination shlinkPagination `json:"pagination"`
			} `json:"visits"`
		}
		query := url.Values{"page": {fmt.Sprint(page)}, "itemsPerPage": {"100"}}
		if item.Domain != nil && *item.Domain != "" {
			query.Set("domain", *item.Domain)
		}
		path := "/rest/v3/short-urls/" + url.PathEscape(item.ShortCode) + "/visits"
		if err := s.fetch(ctx, opts, path, query, &response); err != nil {
			return err
		}

		for _, v := range response.Visits.Data {
			visit := &model.Visit{
				PublicID:   uuid.New().String(),
				ShortURLID: shortURL.ID,
				Referer:    v.Referer,
				UserAgent:  v.UserAgent,
				CreatedAt:  v.Date,
				UpdatedAt:  v.Date,
			}
			if v.VisitLocation != nil {
				visit.Country = v.VisitLocation.CountryCode
			}
			if _, err := s.db.NewInsert().Model(visit).Exec(ctx); err != nil {
				return fmt.Errorf("inserting visit for %q: %w", item.ShortCode, err)
			}
			summary.Visits++
		}

		if page >= response.Visits.Pagination.PagesCount {
			break
		}
	}
	return nil
}
