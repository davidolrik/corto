package middlewares

import (
	"net/http"
	"strings"

	"aidanwoods.dev/go-paseto"
	"github.com/davidolrik/corto/internal/auth"
)

// Auth returns middleware that validates PASETO v4 public tokens from the
// Authorization header. Only paths under /api/ are guarded; everything else
// (short link redirects, docs, OpenAPI spec) is public. Requests to API paths
// starting with any of the skip prefixes are also passed through without
// authentication.
func Auth(publicKey paseto.V4AsymmetricPublicKey, skipPrefixes []string) func(http.Handler) http.Handler {
	parser := paseto.NewParser()
	parser.AddRule(paseto.IssuedBy("corto"))
	parser.AddRule(paseto.NotBeforeNbf())

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only API paths require authentication
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip authentication for specified paths
			for _, prefix := range skipPrefixes {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, `{"title":"Unauthorized","status":401,"detail":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(header, "Bearer ")
			if tokenString == header {
				http.Error(w, `{"title":"Unauthorized","status":401,"detail":"invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			token, err := parser.ParseV4Public(publicKey, tokenString, nil)
			if err != nil {
				http.Error(w, `{"title":"Unauthorized","status":401,"detail":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			userPublicID, err := token.GetSubject()
			if err != nil {
				http.Error(w, `{"title":"Unauthorized","status":401,"detail":"invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			tenantPublicID, err := token.GetString("tenant_id")
			if err != nil {
				http.Error(w, `{"title":"Unauthorized","status":401,"detail":"invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			var isAdmin bool
			if err := token.Get("is_admin", &isAdmin); err != nil {
				http.Error(w, `{"title":"Unauthorized","status":401,"detail":"invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			claims := auth.Claims{
				UserPublicID:   userPublicID,
				TenantPublicID: tenantPublicID,
				IsAdmin:        isAdmin,
			}

			ctx := auth.WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
