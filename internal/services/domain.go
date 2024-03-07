package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/google/uuid"
)

type DomainService struct {
	log *slog.Logger
	db  core.Database
}

func NewDomainService(log *slog.Logger, db core.Database) *DomainService {
	return &DomainService{
		log: log,
		db:  db,
	}
}

func (s *DomainService) CreateDomain(ctx context.Context, d *handlers.DomainData) (*handlers.DomainData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	now := time.Now().Truncate(time.Second)

	domain := &model.Domain{
		PublicID:    uuid.New().String(),
		TenantID:    tenantID,
		FQDN:        d.FQDN,
		FallbackURL: d.FallbackURL,
		Description: d.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err = s.db.NewInsert().Model(domain).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("inserting domain: %w", err)
	}

	return domainToData(domain), nil
}

func (s *DomainService) GetDomain(ctx context.Context, publicID string) (*handlers.DomainData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	domain := &model.Domain{}
	q := s.db.NewSelect().Model(domain).Where("d.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("d.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("domain %q not found: %w", publicID, err)
	}
	return domainToData(domain), nil
}

func (s *DomainService) ListDomains(ctx context.Context) ([]*handlers.DomainData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	var domains []model.Domain
	q := s.db.NewSelect().
		Model(&domains).
		OrderExpr("d.created_at DESC")
	if tenantID != 0 {
		q = q.Where("d.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing domains: %w", err)
	}

	result := make([]*handlers.DomainData, len(domains))
	for i := range domains {
		result[i] = domainToData(&domains[i])
	}
	return result, nil
}

func (s *DomainService) UpdateDomain(ctx context.Context, publicID string, d *handlers.DomainData) (*handlers.DomainData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	existing := &model.Domain{}
	q := s.db.NewSelect().Model(existing).Where("d.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("d.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("domain %q not found: %w", publicID, err)
	}

	now := time.Now().Truncate(time.Second)
	existing.FQDN = d.FQDN
	existing.FallbackURL = d.FallbackURL
	existing.Description = d.Description
	existing.UpdatedAt = now

	_, err = s.db.NewUpdate().Model(existing).WherePK().Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("updating domain: %w", err)
	}

	return domainToData(existing), nil
}

func (s *DomainService) PatchDomain(ctx context.Context, publicID string, patch *handlers.DomainPatch) (*handlers.DomainData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	existing := &model.Domain{}
	q := s.db.NewSelect().Model(existing).Where("d.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("d.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("domain %q not found: %w", publicID, err)
	}

	now := time.Now().Truncate(time.Second)
	columns := []string{"updated_at"}
	existing.UpdatedAt = now

	if patch.FQDN != nil {
		existing.FQDN = *patch.FQDN
		columns = append(columns, "fqdn")
	}
	if patch.FallbackURL != nil {
		existing.FallbackURL = *patch.FallbackURL
		columns = append(columns, "fallback_url")
	}
	if patch.Description != nil {
		existing.Description = *patch.Description
		columns = append(columns, "description")
	}

	_, err = s.db.NewUpdate().Model(existing).Column(columns...).WherePK().Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("patching domain: %w", err)
	}

	return domainToData(existing), nil
}

func (s *DomainService) DeleteDomain(ctx context.Context, publicID string) error {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return err
	}

	domain := &model.Domain{}
	q := s.db.NewSelect().Model(domain).Where("d.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("d.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return fmt.Errorf("domain %q not found: %w", publicID, err)
	}

	// Delete short_urls referencing this domain
	_, err = s.db.NewDelete().Model((*model.ShortURL)(nil)).Where("domain_id = ?", domain.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("deleting short urls: %w", err)
	}

	_, err = s.db.NewDelete().Model(domain).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("deleting domain: %w", err)
	}

	return nil
}

func domainToData(d *model.Domain) *handlers.DomainData {
	return &handlers.DomainData{
		PublicID:    d.PublicID,
		FQDN:        d.FQDN,
		FallbackURL: d.FallbackURL,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
