package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
)

type StatsService struct {
	log *slog.Logger
	db  core.Database
}

func NewStatsService(log *slog.Logger, db core.Database) *StatsService {
	return &StatsService{
		log: log,
		db:  db,
	}
}

// GetStats returns tenant-level statistics: entity counts, visit totals, and
// the per-country visit breakdown. Visits without a resolved country are
// bucketed as "unknown".
func (s *StatsService) GetStats(ctx context.Context) (*handlers.StatsData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	stats := &handlers.StatsData{
		VisitsByCountry: map[string]int{},
	}

	linksQuery := s.db.NewSelect().Model((*model.ShortCode)(nil))
	domainsQuery := s.db.NewSelect().Model((*model.Domain)(nil))
	tagsQuery := s.db.NewSelect().Model((*model.Tag)(nil))
	if tenantID != 0 {
		linksQuery = linksQuery.Where("sc.id IN (?)",
			s.db.NewSelect().Model((*model.ShortURL)(nil)).
				Column("short_code_id").
				Join("JOIN domains AS d ON d.id = su.domain_id").
				Where("d.tenant_id = ?", tenantID))
		domainsQuery = domainsQuery.Where("d.tenant_id = ?", tenantID)
		tagsQuery = tagsQuery.Where("tg.tenant_id = ?", tenantID)
	}
	if stats.Links, err = linksQuery.Count(ctx); err != nil {
		return nil, fmt.Errorf("counting links: %w", err)
	}
	if stats.Domains, err = domainsQuery.Count(ctx); err != nil {
		return nil, fmt.Errorf("counting domains: %w", err)
	}
	if stats.Tags, err = tagsQuery.Count(ctx); err != nil {
		return nil, fmt.Errorf("counting tags: %w", err)
	}

	var visitRows []struct {
		Country  string `bun:"country"`
		Count    int    `bun:"count"`
		ThisWeek int    `bun:"this_week"`
	}
	visitsQuery := s.db.NewSelect().
		Model((*model.Visit)(nil)).
		ColumnExpr("COALESCE(NULLIF(v.country, ''), 'unknown') AS country").
		ColumnExpr("count(*) AS count").
		ColumnExpr("count(*) FILTER (WHERE v.created_at > now() - interval '7 days') AS this_week").
		GroupExpr("COALESCE(NULLIF(v.country, ''), 'unknown')")
	if tenantID != 0 {
		visitsQuery = visitsQuery.
			Join("JOIN short_urls AS su ON su.id = v.short_url_id").
			Join("JOIN domains AS d ON d.id = su.domain_id").
			Where("d.tenant_id = ?", tenantID)
	}
	if err := visitsQuery.Scan(ctx, &visitRows); err != nil {
		return nil, fmt.Errorf("counting visits: %w", err)
	}
	for _, row := range visitRows {
		stats.VisitsByCountry[row.Country] = row.Count
		stats.Visits += row.Count
		stats.VisitsThisWeek += row.ThisWeek
	}

	return stats, nil
}
