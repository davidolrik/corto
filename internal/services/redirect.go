package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/google/uuid"
)

// CountryResolver maps an IP address to an ISO country code. An empty result
// means the country is unknown.
type CountryResolver interface {
	Country(ip string) string
}

type RedirectService struct {
	log       *slog.Logger
	db        core.Database
	countries CountryResolver // nil when GeoIP is not configured
}

func NewRedirectService(log *slog.Logger, db core.Database, countries CountryResolver) *RedirectService {
	return &RedirectService{
		log:       log,
		db:        db,
		countries: countries,
	}
}

// ResolveRedirect looks up the domain by FQDN and the short URL by slug. A
// missing slug on a known domain returns a target with a nil ShortURL so the
// handler can fall back to the domain's fallback URL.
func (s *RedirectService) ResolveRedirect(ctx context.Context, fqdn, slug string) (*handlers.RedirectTarget, error) {
	domain := &model.Domain{}
	err := s.db.NewSelect().Model(domain).
		Where("d.fqdn = ?", fqdn).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("domain %q not found: %w", fqdn, err)
	}

	target := &handlers.RedirectTarget{
		DomainFallbackURL: domain.FallbackURL,
	}

	shortURL := &model.ShortURL{}
	err = s.db.NewSelect().Model(shortURL).
		Relation("ShortCode").
		Relation("ShortCode.PlatformURLs").
		Where("su.domain_id = ?", domain.ID).
		Where("su.slug = ?", slug).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return target, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolving short url %q on %q: %w", slug, fqdn, err)
	}

	shortCode := shortURL.ShortCode
	platformURLs := make([]handlers.RedirectPlatformURL, len(shortCode.PlatformURLs))
	for i, p := range shortCode.PlatformURLs {
		platformURLs[i] = handlers.RedirectPlatformURL{
			Platform:    p.Platform,
			TargetURL:   p.TargetURL,
			FallbackURL: p.FallbackURL,
		}
	}

	target.ShortURL = &handlers.RedirectShortURL{
		PublicID:     shortURL.PublicID,
		TargetURL:    shortCode.TargetURL,
		FallbackURL:  shortCode.FallbackURL,
		ForwardQuery: shortCode.ForwardQuery,
		ValidSince:   shortCode.ValidSince,
		ValidUntil:   shortCode.ValidUntil,
		PlatformURLs: platformURLs,
	}
	return target, nil
}

// RecordVisit stores a single click on a short URL, resolving the visitor's
// country when GeoIP is configured.
func (s *RedirectService) RecordVisit(ctx context.Context, v *handlers.VisitData) error {
	if s.countries != nil && v.Country == "" {
		v.Country = s.countries.Country(v.IPAddress)
	}
	shortURL := &model.ShortURL{}
	err := s.db.NewSelect().Model(shortURL).
		Column("id").
		Where("su.public_id = ?", v.ShortURLPublicID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("short url %q not found: %w", v.ShortURLPublicID, err)
	}

	now := time.Now().Truncate(time.Second)
	visit := &model.Visit{
		PublicID:   uuid.New().String(),
		ShortURLID: shortURL.ID,
		IPAddress:  v.IPAddress,
		UserAgent:  v.UserAgent,
		Referer:    v.Referer,
		Country:    v.Country,
		Campaign:   v.Campaign,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err = s.db.NewInsert().Model(visit).Exec(ctx)
	if err != nil {
		return fmt.Errorf("inserting visit: %w", err)
	}
	return nil
}
