package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/davidolrik/corto/internal/server/handlers"
)

func setupUIMux() *http.ServeMux {
	ui := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>Corto Admin</title>")},
		"assets/app.js": {Data: []byte("console.log('corto')")},
	}
	mux := http.NewServeMux()
	handlers.RegisterUIRoutes(mux, ui)
	return mux
}

func TestUIServesIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	resp := httptest.NewRecorder()
	setupUIMux().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "Corto Admin") {
		t.Errorf("expected index.html content, got: %s", resp.Body.String())
	}
	if ct := resp.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %q", ct)
	}
}

func TestUIServesAssets(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/assets/app.js", nil)
	resp := httptest.NewRecorder()
	setupUIMux().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "console.log") {
		t.Errorf("expected JS content, got: %s", resp.Body.String())
	}
}

func TestUIFallsBackToIndexForClientRoutes(t *testing.T) {
	// Client-side routes like /admin/domains must serve the SPA shell
	req := httptest.NewRequest(http.MethodGet, "/admin/domains", nil)
	resp := httptest.NewRecorder()
	setupUIMux().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "Corto Admin") {
		t.Errorf("expected index.html fallback, got: %s", resp.Body.String())
	}
}

func TestUIRedirectsBareAdminPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp := httptest.NewRecorder()
	setupUIMux().ServeHTTP(resp, req)

	if resp.Code != http.StatusMovedPermanently {
		t.Fatalf("expected status %d, got %d", http.StatusMovedPermanently, resp.Code)
	}
	if loc := resp.Header().Get("Location"); loc != "/admin/" {
		t.Errorf("expected redirect to /admin/, got %q", loc)
	}
}
