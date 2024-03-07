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

type TagService struct {
	log *slog.Logger
	db  core.Database
}

func NewTagService(log *slog.Logger, db core.Database) *TagService {
	return &TagService{
		log: log,
		db:  db,
	}
}

func (s *TagService) CreateTag(ctx context.Context, t *handlers.TagData) (*handlers.TagData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	now := time.Now().Truncate(time.Second)

	tag := &model.Tag{
		PublicID:    uuid.New().String(),
		TenantID:    tenantID,
		Slug:        t.Slug,
		Color:       t.Color,
		Description: t.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err = s.db.NewInsert().Model(tag).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("inserting tag: %w", err)
	}

	return tagToData(tag), nil
}

func (s *TagService) GetTag(ctx context.Context, publicID string) (*handlers.TagData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	tag := &model.Tag{}
	q := s.db.NewSelect().Model(tag).Where("tg.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("tg.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("tag %q not found: %w", publicID, err)
	}
	return tagToData(tag), nil
}

func (s *TagService) ListTags(ctx context.Context) ([]*handlers.TagData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	var tags []model.Tag
	q := s.db.NewSelect().
		Model(&tags).
		OrderExpr("tg.slug ASC")
	if tenantID != 0 {
		q = q.Where("tg.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	result := make([]*handlers.TagData, len(tags))
	for i := range tags {
		result[i] = tagToData(&tags[i])
	}
	return result, nil
}

func (s *TagService) UpdateTag(ctx context.Context, publicID string, t *handlers.TagData) (*handlers.TagData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	existing := &model.Tag{}
	q := s.db.NewSelect().Model(existing).Where("tg.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("tg.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("tag %q not found: %w", publicID, err)
	}

	now := time.Now().Truncate(time.Second)
	existing.Slug = t.Slug
	existing.Color = t.Color
	existing.Description = t.Description
	existing.UpdatedAt = now

	_, err = s.db.NewUpdate().Model(existing).WherePK().Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("updating tag: %w", err)
	}

	return tagToData(existing), nil
}

func (s *TagService) PatchTag(ctx context.Context, publicID string, patch *handlers.TagPatch) (*handlers.TagData, error) {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return nil, err
	}

	existing := &model.Tag{}
	q := s.db.NewSelect().Model(existing).Where("tg.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("tg.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("tag %q not found: %w", publicID, err)
	}

	now := time.Now().Truncate(time.Second)
	columns := []string{"updated_at"}
	existing.UpdatedAt = now

	if patch.Slug != nil {
		existing.Slug = *patch.Slug
		columns = append(columns, "slug")
	}
	if patch.Color != nil {
		existing.Color = *patch.Color
		columns = append(columns, "color")
	}
	if patch.Description != nil {
		existing.Description = *patch.Description
		columns = append(columns, "description")
	}

	_, err = s.db.NewUpdate().Model(existing).Column(columns...).WherePK().Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("patching tag: %w", err)
	}

	return tagToData(existing), nil
}

func (s *TagService) DeleteTag(ctx context.Context, publicID string) error {
	tenantID, err := tenantIDFromContext(ctx, s.db)
	if err != nil {
		return err
	}

	tag := &model.Tag{}
	q := s.db.NewSelect().Model(tag).Where("tg.public_id = ?", publicID)
	if tenantID != 0 {
		q = q.Where("tg.tenant_id = ?", tenantID)
	}
	err = q.Scan(ctx)
	if err != nil {
		return fmt.Errorf("tag %q not found: %w", publicID, err)
	}

	// Remove short_code_tags associations
	_, err = s.db.NewDelete().Model((*model.ShortCodeTag)(nil)).Where("tag_id = ?", tag.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("deleting short code tags: %w", err)
	}

	_, err = s.db.NewDelete().Model(tag).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("deleting tag: %w", err)
	}

	return nil
}

func tagToData(t *model.Tag) *handlers.TagData {
	return &handlers.TagData{
		PublicID:    t.PublicID,
		Slug:        t.Slug,
		Color:       t.Color,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
