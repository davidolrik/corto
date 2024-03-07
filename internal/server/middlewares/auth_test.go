package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/davidolrik/corto/internal/auth"
	"github.com/davidolrik/corto/internal/server/middlewares"
)

func makeTestKey() (paseto.V4AsymmetricSecretKey, paseto.V4AsymmetricPublicKey) {
	sk := paseto.NewV4AsymmetricSecretKey()
	return sk, sk.Public()
}

func makeValidToken(sk paseto.V4AsymmetricSecretKey, userID, tenantID string, isAdmin bool) string {
	now := time.Now()
	token := paseto.NewToken()
	token.SetIssuedAt(now)
	token.SetNotBefore(now)
	token.SetExpiration(now.Add(1 * time.Hour))
	token.SetIssuer("corto")
	token.SetSubject(userID)
	token.SetString("tenant_id", tenantID)
	token.Set("is_admin", isAdmin)
	return token.V4Sign(sk, nil)
}

func makeExpiredToken(sk paseto.V4AsymmetricSecretKey) string {
	past := time.Now().Add(-2 * time.Hour)
	token := paseto.NewToken()
	token.SetIssuedAt(past)
	token.SetNotBefore(past)
	token.SetExpiration(past.Add(1 * time.Hour))
	token.SetIssuer("corto")
	token.SetSubject("user_1")
	token.SetString("tenant_id", "tenant_1")
	token.Set("is_admin", false)
	return token.V4Sign(sk, nil)
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	sk, pk := makeTestKey()
	tokenStr := makeValidToken(sk, "user_1", "tenant_1", true)

	var gotClaims auth.Claims
	handler := middlewares.Auth(pk, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = auth.GetClaims(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	if gotClaims.UserPublicID != "user_1" {
		t.Errorf("expected user_id %q, got %q", "user_1", gotClaims.UserPublicID)
	}
	if gotClaims.TenantPublicID != "tenant_1" {
		t.Errorf("expected tenant_id %q, got %q", "tenant_1", gotClaims.TenantPublicID)
	}
	if !gotClaims.IsAdmin {
		t.Error("expected is_admin to be true")
	}
}

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	_, pk := makeTestKey()

	handler := middlewares.Auth(pk, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestAuthMiddlewareInvalidFormat(t *testing.T) {
	_, pk := makeTestKey()

	handler := middlewares.Auth(pk, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	_, pk := makeTestKey()

	handler := middlewares.Auth(pk, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	req.Header.Set("Authorization", "Bearer v4.public.garbage-token-data")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestAuthMiddlewareExpiredToken(t *testing.T) {
	sk, pk := makeTestKey()
	tokenStr := makeExpiredToken(sk)

	handler := middlewares.Auth(pk, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestAuthMiddlewareWrongKey(t *testing.T) {
	sk, _ := makeTestKey()
	_, otherPK := makeTestKey()
	tokenStr := makeValidToken(sk, "user_1", "tenant_1", false)

	handler := middlewares.Auth(otherPK, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/domains", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestAuthMiddlewareSkipPrefix(t *testing.T) {
	_, pk := makeTestKey()

	called := false
	handler := middlewares.Auth(pk, []string{"/api/auth/"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if !called {
		t.Fatal("expected handler to be called for skipped path")
	}
}

func TestAuthMiddlewareNonAPIPathIsPublic(t *testing.T) {
	_, pk := makeTestKey()

	called := false
	handler := middlewares.Auth(pk, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	// Short link redirects live outside /api and must not require auth
	req := httptest.NewRequest(http.MethodGet, "/my-slug", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if !called {
		t.Fatal("expected handler to be called for non-API path")
	}
}

func TestAuthMiddlewareSkipDocsPath(t *testing.T) {
	_, pk := makeTestKey()

	called := false
	handler := middlewares.Auth(pk, []string{"/api/auth/", "/openapi", "/docs"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if !called {
		t.Fatal("expected handler to be called for docs path")
	}
}
