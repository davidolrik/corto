package auth

import "context"

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	tenantIDKey contextKey = "tenant_id"
	isAdminKey  contextKey = "is_admin"
)

// Claims holds the authenticated user's claims extracted from a token.
type Claims struct {
	UserPublicID   string
	TenantPublicID string
	IsAdmin        bool
}

// WithClaims returns a new context with the given claims.
func WithClaims(ctx context.Context, claims Claims) context.Context {
	ctx = context.WithValue(ctx, userIDKey, claims.UserPublicID)
	ctx = context.WithValue(ctx, tenantIDKey, claims.TenantPublicID)
	ctx = context.WithValue(ctx, isAdminKey, claims.IsAdmin)
	return ctx
}

// GetClaims extracts claims from the context. Returns zero Claims if not present.
func GetClaims(ctx context.Context) Claims {
	var c Claims
	if v, ok := ctx.Value(userIDKey).(string); ok {
		c.UserPublicID = v
	}
	if v, ok := ctx.Value(tenantIDKey).(string); ok {
		c.TenantPublicID = v
	}
	if v, ok := ctx.Value(isAdminKey).(bool); ok {
		c.IsAdmin = v
	}
	return c
}
