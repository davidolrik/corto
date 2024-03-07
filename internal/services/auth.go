package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/davidolrik/corto/internal/auth"
	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/zitadel/passwap"
	"github.com/zitadel/passwap/bcrypt"
)

type AuthService struct {
	log       *slog.Logger
	db        core.Database
	secretKey paseto.V4AsymmetricSecretKey
	passwords *passwap.Swapper
}

func NewAuthService(log *slog.Logger, db core.Database, secretKeyHex string) (*AuthService, error) {
	secretKey, err := paseto.NewV4AsymmetricSecretKeyFromHex(secretKeyHex)
	if err != nil {
		return nil, fmt.Errorf("parsing secret key: %w", err)
	}
	passwords := passwap.NewSwapper(
		bcrypt.New(bcrypt.DefaultCost, nil),
	)
	return &AuthService{
		log:       log,
		db:        db,
		secretKey: secretKey,
		passwords: passwords,
	}, nil
}

// tenantMembership is one tenant a user has access to, with admin status.
type tenantMembership struct {
	PublicID string `bun:"public_id"`
	Slug     string `bun:"slug"`
	Name     string `bun:"name"`
	IsAdmin  bool   `bun:"is_admin"`
}

// memberships lists the tenants a user has access to, ordered by name.
func (s *AuthService) memberships(ctx context.Context, userID int) ([]tenantMembership, error) {
	var rows []tenantMembership
	err := s.db.NewSelect().
		TableExpr("tenants AS t").
		ColumnExpr("t.public_id, t.slug, t.name, tua.is_admin").
		Join("JOIN tenant_user_access AS tua ON tua.tenant_id = t.id").
		Where("tua.user_id = ?", userID).
		OrderExpr("t.name ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("listing tenant memberships: %w", err)
	}
	return rows, nil
}

// loginResult mints a token for the chosen tenant and assembles the result.
func (s *AuthService) loginResult(user *model.User, active tenantMembership, all []tenantMembership) (*handlers.LoginResult, error) {
	now := time.Now()
	token := paseto.NewToken()
	token.SetIssuedAt(now)
	token.SetNotBefore(now)
	token.SetExpiration(now.Add(24 * time.Hour))
	token.SetIssuer("corto")
	token.SetSubject(user.PublicID)
	token.SetString("tenant_id", active.PublicID)
	if err := token.Set("is_admin", active.IsAdmin); err != nil {
		return nil, fmt.Errorf("setting is_admin claim: %w", err)
	}

	tenants := make([]handlers.TenantMembership, len(all))
	for i, m := range all {
		tenants[i] = handlers.TenantMembership{Slug: m.Slug, Name: m.Name, IsAdmin: m.IsAdmin}
	}

	return &handlers.LoginResult{
		Token:          token.V4Sign(s.secretKey, nil),
		UserPublicID:   user.PublicID,
		Username:       user.Username,
		TenantPublicID: active.PublicID,
		TenantSlug:     active.Slug,
		TenantName:     active.Name,
		IsAdmin:        active.IsAdmin,
		Tenants:        tenants,
	}, nil
}

// chooseTenant picks the membership matching the slug, or the first
// membership when no slug is given.
func chooseTenant(all []tenantMembership, slug string) (tenantMembership, error) {
	if len(all) == 0 {
		return tenantMembership{}, fmt.Errorf("no tenant access")
	}
	if slug == "" {
		return all[0], nil
	}
	for _, m := range all {
		if m.Slug == slug {
			return m, nil
		}
	}
	return tenantMembership{}, fmt.Errorf("no access to tenant %q", slug)
}

// Login authenticates a user. The tenant slug is optional; without it the
// user's first tenant becomes the active one.
func (s *AuthService) Login(ctx context.Context, username, password, tenantSlug string) (*handlers.LoginResult, error) {
	// Look up user
	user := &model.User{}
	err := s.db.NewSelect().Model(user).Where("username = ?", username).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Verify password and upgrade hash if needed
	updated, err := s.passwords.Verify(user.Password, password)
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}
	if updated != "" {
		user.Password = updated
		_, err = s.db.NewUpdate().Model(user).Column("password").WherePK().Exec(ctx)
		if err != nil {
			s.log.Warn("Failed to update password hash", "user_id", user.PublicID, "error", err)
		}
	}

	all, err := s.memberships(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	active, err := chooseTenant(all, tenantSlug)
	if err != nil {
		return nil, err
	}

	return s.loginResult(user, active, all)
}

// SwitchTenant mints a new token for another tenant the authenticated user
// has access to.
func (s *AuthService) SwitchTenant(ctx context.Context, tenantSlug string) (*handlers.LoginResult, error) {
	claims := auth.GetClaims(ctx)
	if claims.UserPublicID == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	user := &model.User{}
	err := s.db.NewSelect().Model(user).Where("public_id = ?", claims.UserPublicID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	all, err := s.memberships(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	active, err := chooseTenant(all, tenantSlug)
	if err != nil {
		return nil, err
	}

	return s.loginResult(user, active, all)
}
