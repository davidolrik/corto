package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"
)

// pgerrUniqueViolation is the PostgreSQL SQLSTATE for unique constraint violations.
const pgerrUniqueViolation = "23505"

type ShortCodeService struct {
	log *slog.Logger
	db  core.Database
}

func NewShortCodeService(log *slog.Logger, db core.Database) *ShortCodeService {
	return &ShortCodeService{
		log: log,
		db:  db,
	}
}

const (
	slugAlphabet        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	generatedSlugLength = 7
	slugAttempts        = 5
)

// generateSlug returns a random base62 short code.
func generateSlug() (string, error) {
	slug := make([]byte, generatedSlugLength)
	for i := range slug {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(slugAlphabet))))
		if err != nil {
			return "", fmt.Errorf("generating slug: %w", err)
		}
		slug[i] = slugAlphabet[n.Int64()]
	}
	return string(slug), nil
}

func (s *ShortCodeService) CreateShortCode(ctx context.Context, sc *handlers.ShortCodeData) (*handlers.ShortCodeData, error) {
	now := time.Now().Truncate(time.Second)

	// Without a slug, generate a random one, retrying on the unlikely
	// collision with an existing slug on one of the domains.
	generateOnCreate := sc.Slug == ""
	for attempt := 1; ; attempt++ {
		if generateOnCreate {
			slug, err := generateSlug()
			if err != nil {
				return nil, err
			}
			sc.Slug = slug
		}

		shortCode := &model.ShortCode{
			PublicID:     uuid.New().String(),
			Title:        sc.Title,
			Description:  sc.Description,
			Slug:         sc.Slug,
			TargetURL:    sc.TargetURL,
			FallbackURL:  sc.FallbackURL,
			IsCrawlable:  sc.IsCrawlable,
			ForwardQuery: sc.ForwardQuery,
			ValidSince:   sc.ValidSince,
			ValidUntil:   sc.ValidUntil,
			MaxVisits:    sc.MaxVisits,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.NewInsert().Model(shortCode).Exec(ctx); err != nil {
				return fmt.Errorf("inserting short code: %w", err)
			}
			if err := s.syncDomains(ctx, tx, shortCode.ID, shortCode.Slug, sc.Domains, now); err != nil {
				return err
			}
			return s.syncTags(ctx, tx, shortCode.ID, sc.Tags, now)
		})
		if err == nil {
			sc.PublicID = shortCode.PublicID
			break
		}
		if generateOnCreate && errors.Is(err, handlers.ErrConflict) && attempt < slugAttempts {
			continue
		}
		return nil, err
	}

	sc.CreatedAt = now
	sc.UpdatedAt = now
	return sc, nil
}

func (s *ShortCodeService) GetShortCode(ctx context.Context, publicID string) (*handlers.ShortCodeData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	shortCode := &model.ShortCode{}
	q := s.db.NewSelect().
		Model(shortCode).
		Relation("ShortURLs", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Relation("Domain")
		}).
		Relation("Tags").
		Where("sc.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("sc.id IN (?)",
			s.db.NewSelect().Model((*model.ShortURL)(nil)).
				Column("short_code_id").
				Join("JOIN domains AS d ON d.id = su.domain_id").
				Where("d.tenant_id = ?", tenantID))
	}
	err = q.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("short code %q %w", publicID, handlers.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("loading short code %q: %w", publicID, err)
	}

	counts, err := s.visitCounts(ctx, []int{shortCode.ID})
	if err != nil {
		return nil, err
	}
	campaigns, err := s.campaignCounts(ctx, []int{shortCode.ID})
	if err != nil {
		return nil, err
	}
	countries, err := s.countryCounts(ctx, []int{shortCode.ID})
	if err != nil {
		return nil, err
	}

	data := shortCodeToData(shortCode)
	applyVisitCounts(data, counts[shortCode.ID])
	data.VisitsByCampaign = campaigns[shortCode.ID]
	data.VisitsByCountry = countries[shortCode.ID]
	return data, nil
}

func (s *ShortCodeService) ListShortCodes(ctx context.Context) ([]*handlers.ShortCodeData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	var shortCodes []model.ShortCode
	q := s.db.NewSelect().
		Model(&shortCodes).
		Relation("ShortURLs", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Relation("Domain")
		}).
		Relation("Tags").
		OrderExpr("sc.created_at DESC")
	if tenantID != 0 {
		q = q.Where("sc.id IN (?)",
			s.db.NewSelect().Model((*model.ShortURL)(nil)).
				Column("short_code_id").
				Join("JOIN domains AS d ON d.id = su.domain_id").
				Where("d.tenant_id = ?", tenantID))
	}
	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing short codes: %w", err)
	}

	ids := make([]int, len(shortCodes))
	for i := range shortCodes {
		ids[i] = shortCodes[i].ID
	}
	counts, err := s.visitCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	campaigns, err := s.campaignCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	countries, err := s.countryCounts(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]*handlers.ShortCodeData, len(shortCodes))
	for i := range shortCodes {
		result[i] = shortCodeToData(&shortCodes[i])
		applyVisitCounts(result[i], counts[shortCodes[i].ID])
		result[i].VisitsByCampaign = campaigns[shortCodes[i].ID]
		result[i].VisitsByCountry = countries[shortCodes[i].ID]
	}
	return result, nil
}

func (s *ShortCodeService) UpdateShortCode(ctx context.Context, publicID string, sc *handlers.ShortCodeData) (*handlers.ShortCodeData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	existing := &model.ShortCode{}
	q := s.db.NewSelect().Model(existing).Where("sc.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("sc.id IN (?)",
			s.db.NewSelect().Model((*model.ShortURL)(nil)).
				Column("short_code_id").
				Join("JOIN domains AS d ON d.id = su.domain_id").
				Where("d.tenant_id = ?", tenantID))
	}
	err = q.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("short code %q %w", publicID, handlers.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("loading short code %q: %w", publicID, err)
	}

	now := time.Now().Truncate(time.Second)
	existing.Title = sc.Title
	existing.Description = sc.Description
	existing.Slug = sc.Slug
	existing.TargetURL = sc.TargetURL
	existing.FallbackURL = sc.FallbackURL
	existing.IsCrawlable = sc.IsCrawlable
	existing.ForwardQuery = sc.ForwardQuery
	existing.ValidSince = sc.ValidSince
	existing.ValidUntil = sc.ValidUntil
	existing.MaxVisits = sc.MaxVisits
	existing.UpdatedAt = now

	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Model(existing).WherePK().Exec(ctx); err != nil {
			return fmt.Errorf("updating short code: %w", err)
		}
		if err := s.syncDomains(ctx, tx, existing.ID, existing.Slug, sc.Domains, now); err != nil {
			return err
		}
		return s.syncTags(ctx, tx, existing.ID, sc.Tags, now)
	})
	if err != nil {
		return nil, err
	}

	sc.PublicID = publicID
	sc.CreatedAt = existing.CreatedAt
	sc.UpdatedAt = now
	return sc, nil
}

func (s *ShortCodeService) PatchShortCode(ctx context.Context, publicID string, patch *handlers.ShortCodePatch) (*handlers.ShortCodeData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	existing := &model.ShortCode{}
	q := s.db.NewSelect().Model(existing).Where("sc.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("sc.id IN (?)",
			s.db.NewSelect().Model((*model.ShortURL)(nil)).
				Column("short_code_id").
				Join("JOIN domains AS d ON d.id = su.domain_id").
				Where("d.tenant_id = ?", tenantID))
	}
	err = q.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("short code %q %w", publicID, handlers.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("loading short code %q: %w", publicID, err)
	}

	now := time.Now().Truncate(time.Second)
	columns := []string{"updated_at"}
	existing.UpdatedAt = now

	if patch.Title != nil {
		existing.Title = *patch.Title
		columns = append(columns, "title")
	}
	if patch.Description != nil {
		existing.Description = *patch.Description
		columns = append(columns, "description")
	}
	if patch.Slug != nil {
		existing.Slug = *patch.Slug
		columns = append(columns, "slug")
	}
	if patch.TargetURL != nil {
		existing.TargetURL = *patch.TargetURL
		columns = append(columns, "target_url")
	}
	if patch.FallbackURL != nil {
		existing.FallbackURL = *patch.FallbackURL
		columns = append(columns, "fallback_url")
	}
	if patch.IsCrawlable != nil {
		existing.IsCrawlable = *patch.IsCrawlable
		columns = append(columns, "is_crawlable")
	}
	if patch.ForwardQuery != nil {
		existing.ForwardQuery = *patch.ForwardQuery
		columns = append(columns, "forward_query")
	}
	if patch.ValidSince != nil {
		existing.ValidSince = patch.ValidSince
		columns = append(columns, "valid_since")
	}
	if patch.ValidUntil != nil {
		existing.ValidUntil = patch.ValidUntil
		columns = append(columns, "valid_until")
	}
	if patch.MaxVisits != nil {
		existing.MaxVisits = patch.MaxVisits
		columns = append(columns, "max_visits")
	}

	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model(existing).Column(columns...).WherePK().Exec(ctx)
		if err != nil {
			return fmt.Errorf("patching short code: %w", err)
		}

		if patch.Domains != nil {
			if err := s.syncDomains(ctx, tx, existing.ID, existing.Slug, *patch.Domains, now); err != nil {
				return err
			}
		} else if patch.Slug != nil {
			// Keep the denormalized slug on short_urls in sync with the rename
			_, err = tx.NewUpdate().Model((*model.ShortURL)(nil)).
				Set("slug = ?", existing.Slug).
				Set("updated_at = ?", now).
				Where("short_code_id = ?", existing.ID).
				Exec(ctx)
			if errIsUniqueViolation(err) {
				return fmt.Errorf("slug %q is already in use on one of the link's domains: %w", existing.Slug, handlers.ErrConflict)
			}
			if err != nil {
				return fmt.Errorf("updating short url slugs: %w", err)
			}
		}

		if patch.Tags != nil {
			if err := s.syncTags(ctx, tx, existing.ID, *patch.Tags, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Re-fetch with relations to return complete data
	return s.GetShortCode(ctx, publicID)
}

func (s *ShortCodeService) DeleteShortCode(ctx context.Context, publicID string) error {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return err
	}

	shortCode := &model.ShortCode{}
	q := s.db.NewSelect().Model(shortCode).Where("sc.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("sc.id IN (?)",
			s.db.NewSelect().Model((*model.ShortURL)(nil)).
				Column("short_code_id").
				Join("JOIN domains AS d ON d.id = su.domain_id").
				Where("d.tenant_id = ?", tenantID))
	}
	err = q.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("short code %q %w", publicID, handlers.ErrNotFound)
	}
	if err != nil {
		return fmt.Errorf("loading short code %q: %w", publicID, err)
	}

	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Delete related short_code_tags
		_, err := tx.NewDelete().Model((*model.ShortCodeTag)(nil)).Where("shortcode_id = ?", shortCode.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("deleting short code tags: %w", err)
		}

		// Delete visits via short_urls
		_, err = tx.NewDelete().Model((*model.Visit)(nil)).
			Where("short_url_id IN (?)", tx.NewSelect().Model((*model.ShortURL)(nil)).Column("id").Where("short_code_id = ?", shortCode.ID)).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("deleting visits: %w", err)
		}

		// Delete short_urls
		_, err = tx.NewDelete().Model((*model.ShortURL)(nil)).Where("short_code_id = ?", shortCode.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("deleting short urls: %w", err)
		}

		// Delete the short code itself
		_, err = tx.NewDelete().Model(shortCode).WherePK().Exec(ctx)
		if err != nil {
			return fmt.Errorf("deleting short code: %w", err)
		}

		return nil
	})
}

// syncDomains reconciles the short_urls of a short code with the given domain
// FQDNs. Unchanged associations are kept so their visit history and public IDs
// survive edits; removed associations are deleted together with their visits.
// When tenant-scoped, verifies each domain belongs to the tenant.
func (s *ShortCodeService) syncDomains(ctx context.Context, idb bun.IDB, shortCodeID int, slug string, fqdns []string, now time.Time) error {
	tenantID, err := tenantIDFromContext(ctx, idb)
	if err != nil {
		return err
	}

	var existing []model.ShortURL
	err = idb.NewSelect().Model(&existing).Where("su.short_code_id = ?", shortCodeID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("loading short urls: %w", err)
	}
	existingByDomain := make(map[int]*model.ShortURL, len(existing))
	for i := range existing {
		existingByDomain[existing[i].DomainID] = &existing[i]
	}

	desired := make(map[int]bool, len(fqdns))
	for _, fqdn := range fqdns {
		domain := &model.Domain{}
		q := idb.NewSelect().Model(domain).Where("fqdn = ?", fqdn)
		if tenantID != 0 {
			q = q.Where("tenant_id = ?", tenantID)
		}
		if err := q.Scan(ctx); err != nil {
			return fmt.Errorf("domain %q %w", fqdn, handlers.ErrNotFound)
		}
		desired[domain.ID] = true

		if shortURL, ok := existingByDomain[domain.ID]; ok {
			if shortURL.Slug != slug {
				shortURL.Slug = slug
				shortURL.UpdatedAt = now
				_, err := idb.NewUpdate().Model(shortURL).Column("slug", "updated_at").WherePK().Exec(ctx)
				if errIsUniqueViolation(err) {
					return fmt.Errorf("slug %q is already in use on domain %q: %w", slug, fqdn, handlers.ErrConflict)
				}
				if err != nil {
					return fmt.Errorf("updating short url for domain %q: %w", fqdn, err)
				}
			}
			continue
		}

		shortURL := &model.ShortURL{
			PublicID:    uuid.New().String(),
			DomainID:    domain.ID,
			ShortCodeID: shortCodeID,
			Slug:        slug,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		_, err := idb.NewInsert().Model(shortURL).Exec(ctx)
		if errIsUniqueViolation(err) {
			return fmt.Errorf("slug %q is already in use on domain %q: %w", slug, fqdn, handlers.ErrConflict)
		}
		if err != nil {
			return fmt.Errorf("inserting short url for domain %q: %w", fqdn, err)
		}
	}

	// Remove associations to domains no longer listed, along with their
	// visit history which references them.
	for domainID, shortURL := range existingByDomain {
		if desired[domainID] {
			continue
		}
		_, err := idb.NewDelete().Model((*model.Visit)(nil)).Where("short_url_id = ?", shortURL.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("deleting visits for removed domain: %w", err)
		}
		_, err = idb.NewDelete().Model(shortURL).WherePK().Exec(ctx)
		if err != nil {
			return fmt.Errorf("deleting short url for removed domain: %w", err)
		}
	}
	return nil
}

// domainVisitCount holds visit totals for one domain of a short code.
type domainVisitCount struct {
	Total    int
	ThisWeek int
}

// visitCounts returns the number of visits per short code ID, broken down by
// domain FQDN, with totals for all time and the last 7 days.
func (s *ShortCodeService) visitCounts(ctx context.Context, shortCodeIDs []int) (map[int]map[string]domainVisitCount, error) {
	if len(shortCodeIDs) == 0 {
		return map[int]map[string]domainVisitCount{}, nil
	}

	var rows []struct {
		ShortCodeID int    `bun:"short_code_id"`
		FQDN        string `bun:"fqdn"`
		Count       int    `bun:"count"`
		ThisWeek    int    `bun:"this_week"`
	}
	err := s.db.NewSelect().
		Model((*model.Visit)(nil)).
		ColumnExpr("su.short_code_id").
		ColumnExpr("d.fqdn").
		ColumnExpr("count(*) AS count").
		ColumnExpr("count(*) FILTER (WHERE v.created_at > now() - interval '7 days') AS this_week").
		Join("JOIN short_urls AS su ON su.id = v.short_url_id").
		Join("JOIN domains AS d ON d.id = su.domain_id").
		Where("su.short_code_id IN (?)", bun.In(shortCodeIDs)).
		GroupExpr("su.short_code_id, d.fqdn").
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("counting visits: %w", err)
	}

	counts := make(map[int]map[string]domainVisitCount, len(rows))
	for _, row := range rows {
		if counts[row.ShortCodeID] == nil {
			counts[row.ShortCodeID] = map[string]domainVisitCount{}
		}
		counts[row.ShortCodeID][row.FQDN] = domainVisitCount{Total: row.Count, ThisWeek: row.ThisWeek}
	}
	return counts, nil
}

// visitBreakdown returns the number of visits per short code ID, grouped by
// the given SQL dimension expression (e.g. campaign or country).
func (s *ShortCodeService) visitBreakdown(ctx context.Context, shortCodeIDs []int, dimension string) (map[int]map[string]int, error) {
	if len(shortCodeIDs) == 0 {
		return map[int]map[string]int{}, nil
	}

	var rows []struct {
		ShortCodeID int    `bun:"short_code_id"`
		Bucket      string `bun:"bucket"`
		Count       int    `bun:"count"`
	}
	err := s.db.NewSelect().
		Model((*model.Visit)(nil)).
		ColumnExpr("su.short_code_id").
		ColumnExpr(dimension+" AS bucket").
		ColumnExpr("count(*) AS count").
		Join("JOIN short_urls AS su ON su.id = v.short_url_id").
		Where("su.short_code_id IN (?)", bun.In(shortCodeIDs)).
		GroupExpr("su.short_code_id, "+dimension).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("counting visits by %s: %w", dimension, err)
	}

	counts := make(map[int]map[string]int, len(rows))
	for _, row := range rows {
		if counts[row.ShortCodeID] == nil {
			counts[row.ShortCodeID] = map[string]int{}
		}
		counts[row.ShortCodeID][row.Bucket] = row.Count
	}
	return counts, nil
}

// campaignCounts breaks down visits by campaign; visits without a campaign
// are bucketed as "direct".
func (s *ShortCodeService) campaignCounts(ctx context.Context, shortCodeIDs []int) (map[int]map[string]int, error) {
	return s.visitBreakdown(ctx, shortCodeIDs, "COALESCE(NULLIF(v.campaign, ''), 'direct')")
}

// countryCounts breaks down visits by country; visits without a resolved
// country are bucketed as "unknown".
func (s *ShortCodeService) countryCounts(ctx context.Context, shortCodeIDs []int) (map[int]map[string]int, error) {
	return s.visitBreakdown(ctx, shortCodeIDs, "COALESCE(NULLIF(v.country, ''), 'unknown')")
}

// applyVisitCounts sets the per-domain breakdown and totals on a short code.
func applyVisitCounts(data *handlers.ShortCodeData, byDomain map[string]domainVisitCount) {
	data.VisitsByDomain = make(map[string]int, len(byDomain))
	data.Visits = 0
	data.VisitsThisWeek = 0
	for fqdn, count := range byDomain {
		data.VisitsByDomain[fqdn] = count.Total
		data.Visits += count.Total
		data.VisitsThisWeek += count.ThisWeek
	}
}

// errIsUniqueViolation reports whether err is a PostgreSQL unique constraint
// violation.
func errIsUniqueViolation(err error) bool {
	var pgErr pgdriver.Error
	return errors.As(err, &pgErr) && pgErr.Field('C') == pgerrUniqueViolation
}

// syncTags replaces all short_code_tags for a short code with the given tag slugs.
// When tenant-scoped, verifies each tag belongs to the tenant.
func (s *ShortCodeService) syncTags(ctx context.Context, idb bun.IDB, shortCodeID int, slugs []string, now time.Time) error {
	tenantID, err := tenantIDFromContext(ctx, idb)
	if err != nil {
		return err
	}

	// Remove existing tag associations
	_, err = idb.NewDelete().Model((*model.ShortCodeTag)(nil)).Where("shortcode_id = ?", shortCodeID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("clearing short code tags: %w", err)
	}

	for _, slug := range slugs {
		tag := &model.Tag{}
		q := idb.NewSelect().Model(tag).Where("slug = ?", slug)
		if tenantID != 0 {
			q = q.Where("tenant_id = ?", tenantID)
		}
		err := q.Scan(ctx)
		if err != nil {
			return fmt.Errorf("tag %q %w", slug, handlers.ErrNotFound)
		}
		sct := &model.ShortCodeTag{
			TagID:       tag.ID,
			ShortCodeID: shortCodeID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		_, err = idb.NewInsert().Model(sct).Exec(ctx)
		if err != nil {
			return fmt.Errorf("inserting short code tag %q: %w", slug, err)
		}
	}
	return nil
}

func shortCodeToData(sc *model.ShortCode) *handlers.ShortCodeData {
	domains := make([]string, 0, len(sc.ShortURLs))
	for _, su := range sc.ShortURLs {
		if su.Domain != nil {
			domains = append(domains, su.Domain.FQDN)
		}
	}

	tags := make([]string, 0, len(sc.Tags))
	for _, t := range sc.Tags {
		tags = append(tags, t.Slug)
	}

	return &handlers.ShortCodeData{
		PublicID:     sc.PublicID,
		Title:        sc.Title,
		Description:  sc.Description,
		Slug:         sc.Slug,
		TargetURL:    sc.TargetURL,
		FallbackURL:  sc.FallbackURL,
		IsCrawlable:  sc.IsCrawlable,
		ForwardQuery: sc.ForwardQuery,
		ValidSince:   sc.ValidSince,
		ValidUntil:   sc.ValidUntil,
		MaxVisits:    sc.MaxVisits,
		Domains:      domains,
		Tags:         tags,
		CreatedAt:    sc.CreatedAt,
		UpdatedAt:    sc.UpdatedAt,
	}
}
