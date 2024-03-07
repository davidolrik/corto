package services

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/davidolrik/corto/internal/auth"
	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TenantService struct {
	log *slog.Logger
	db  core.Database
}

func NewTenantService(log *slog.Logger, db core.Database) *TenantService {
	return &TenantService{
		log: log,
		db:  db,
	}
}

var slugifyPattern = regexp.MustCompile(`[^a-z0-9]+`)

// slugify derives a URL safe slug from a name.
func slugify(name string) string {
	return strings.Trim(slugifyPattern.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// CreateTenant creates a tenant owned by the given user and grants the owner
// admin access to it. An empty slug is derived from the name.
func (s *TenantService) CreateTenant(ctx context.Context, name, slug, ownerUsername string) (*model.Tenant, error) {
	if name == "" {
		return nil, fmt.Errorf("tenant name must not be empty")
	}
	if slug == "" {
		slug = slugify(name)
	}

	owner := &model.User{}
	err := s.db.NewSelect().Model(owner).
		Where("username = ?", ownerUsername).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("owner %q not found: %w", ownerUsername, err)
	}

	now := time.Now().Truncate(time.Second)
	tenant := &model.Tenant{
		PublicID:  uuid.New().String(),
		OwnerID:   owner.ID,
		Slug:      slug,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(tenant).Exec(ctx); err != nil {
			if errIsUniqueViolation(err) {
				return fmt.Errorf("tenant slug %q is already taken: %w", slug, handlers.ErrConflict)
			}
			return fmt.Errorf("inserting tenant: %w", err)
		}
		access := &model.TenantUserAccess{
			TenantID:  tenant.ID,
			UserID:    owner.ID,
			IsAdmin:   true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(access).Exec(ctx); err != nil {
			return fmt.Errorf("granting owner access: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

// tenantIDFromContext resolves the tenant's internal ID from the auth claims in
// the context. Returns 0 and nil error if no claims are present (for
// unauthenticated contexts like tests). Accepts any bun.IDB so it works both
// on plain connections and inside transactions.
func tenantIDFromContext(ctx context.Context, db bun.IDB) (int, error) {
	claims := auth.GetClaims(ctx)
	if claims.TenantPublicID == "" {
		return 0, nil
	}

	tenant := &model.Tenant{}
	err := db.NewSelect().Model(tenant).
		Column("id").
		Where("public_id = ?", claims.TenantPublicID).
		Scan(ctx)
	if err != nil {
		return 0, fmt.Errorf("tenant %q not found: %w", claims.TenantPublicID, err)
	}
	return tenant.ID, nil
}
