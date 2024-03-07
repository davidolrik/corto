package handlers_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/davidolrik/corto/internal/server/handlers"
)

type fakeRedirectDomain struct {
	fallbackURL string
	shortURLs   map[string]*handlers.RedirectShortURL
}

type fakeRedirectStore struct {
	domains  map[string]*fakeRedirectDomain
	visits   []*handlers.VisitData
	visitErr error
}

func newFakeRedirectStore() *fakeRedirectStore {
	return &fakeRedirectStore{
		domains: make(map[string]*fakeRedirectDomain),
	}
}

func (s *fakeRedirectStore) addDomain(fqdn, fallbackURL string) *fakeRedirectDomain {
	d := &fakeRedirectDomain{
		fallbackURL: fallbackURL,
		shortURLs:   make(map[string]*handlers.RedirectShortURL),
	}
	s.domains[fqdn] = d
	return d
}

func (s *fakeRedirectStore) ResolveRedirect(_ context.Context, fqdn, slug string) (*handlers.RedirectTarget, error) {
	d, ok := s.domains[fqdn]
	if !ok {
		return nil, fmt.Errorf("domain %q not found", fqdn)
	}
	return &handlers.RedirectTarget{
		DomainFallbackURL: d.fallbackURL,
		ShortURL:          d.shortURLs[slug],
	}, nil
}

func (s *fakeRedirectStore) RecordVisit(_ context.Context, v *handlers.VisitData) error {
	if s.visitErr != nil {
		return s.visitErr
	}
	s.visits = append(s.visits, v)
	return nil
}

func setupRedirectMux(store handlers.RedirectStore) *http.ServeMux {
	mux := http.NewServeMux()
	handlers.RegisterRedirectRoutes(mux, store)
	return mux
}

func doRedirectRequest(mux *http.ServeMux, url string, header http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	return resp
}

func assertRedirect(t *testing.T, resp *httptest.ResponseRecorder, location string) {
	t.Helper()
	if resp.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusFound, resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Location"); got != location {
		t.Fatalf("expected Location %q, got %q", location, got)
	}
}

func TestRedirectHappyPath(t *testing.T) {
	store := newFakeRedirectStore()
	store.addDomain("go.example.com", "").shortURLs["promo"] = &handlers.RedirectShortURL{
		PublicID:  "su_1",
		TargetURL: "https://example.com/landing",
	}

	resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/promo", nil)

	assertRedirect(t, resp, "https://example.com/landing")
}

func TestRedirectStripsHostPort(t *testing.T) {
	store := newFakeRedirectStore()
	store.addDomain("go.example.com", "").shortURLs["promo"] = &handlers.RedirectShortURL{
		PublicID:  "su_1",
		TargetURL: "https://example.com/landing",
	}

	resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com:8080/promo", nil)

	assertRedirect(t, resp, "https://example.com/landing")
}

func TestRedirectUnknownDomain(t *testing.T) {
	store := newFakeRedirectStore()

	resp := doRedirectRequest(setupRedirectMux(store), "http://unknown.example.com/promo", nil)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestRedirectUnknownSlugWithDomainFallback(t *testing.T) {
	store := newFakeRedirectStore()
	store.addDomain("go.example.com", "https://example.com/not-found")

	resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/nope", nil)

	assertRedirect(t, resp, "https://example.com/not-found")
}

func TestRedirectUnknownSlugWithoutDomainFallback(t *testing.T) {
	store := newFakeRedirectStore()
	store.addDomain("go.example.com", "")

	resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/nope", nil)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestRedirectValidityWindow(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	cases := []struct {
		name           string
		shortURL       *handlers.RedirectShortURL
		domainFallback string
		wantStatus     int
		wantLocation   string
	}{
		{
			name: "within window redirects to target",
			shortURL: &handlers.RedirectShortURL{
				PublicID:   "su_1",
				TargetURL:  "https://example.com/landing",
				ValidSince: &past,
				ValidUntil: &future,
			},
			wantStatus:   http.StatusFound,
			wantLocation: "https://example.com/landing",
		},
		{
			name: "not yet valid uses short code fallback",
			shortURL: &handlers.RedirectShortURL{
				PublicID:    "su_1",
				TargetURL:   "https://example.com/landing",
				FallbackURL: "https://example.com/coming-soon",
				ValidSince:  &future,
			},
			wantStatus:   http.StatusFound,
			wantLocation: "https://example.com/coming-soon",
		},
		{
			name: "expired uses short code fallback",
			shortURL: &handlers.RedirectShortURL{
				PublicID:    "su_1",
				TargetURL:   "https://example.com/landing",
				FallbackURL: "https://example.com/expired",
				ValidUntil:  &past,
			},
			wantStatus:   http.StatusFound,
			wantLocation: "https://example.com/expired",
		},
		{
			name: "expired without short code fallback uses domain fallback",
			shortURL: &handlers.RedirectShortURL{
				PublicID:   "su_1",
				TargetURL:  "https://example.com/landing",
				ValidUntil: &past,
			},
			domainFallback: "https://example.com/not-found",
			wantStatus:     http.StatusFound,
			wantLocation:   "https://example.com/not-found",
		},
		{
			name: "expired without any fallback is not found",
			shortURL: &handlers.RedirectShortURL{
				PublicID:   "su_1",
				TargetURL:  "https://example.com/landing",
				ValidUntil: &past,
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeRedirectStore()
			store.addDomain("go.example.com", tc.domainFallback).shortURLs["promo"] = tc.shortURL

			resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/promo", nil)

			if tc.wantStatus == http.StatusFound {
				assertRedirect(t, resp, tc.wantLocation)
			} else if resp.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, resp.Code)
			}
		})
	}
}

func TestRedirectForwardQuery(t *testing.T) {
	cases := []struct {
		name         string
		targetURL    string
		forwardQuery bool
		url          string
		wantLocation string
	}{
		{
			name:         "forwards query string to target",
			targetURL:    "https://example.com/landing",
			forwardQuery: true,
			url:          "http://go.example.com/promo?utm_campaign=spring&ref=mail",
			wantLocation: "https://example.com/landing?utm_campaign=spring&ref=mail",
		},
		{
			name:         "appends to existing target query",
			targetURL:    "https://example.com/landing?lang=en",
			forwardQuery: true,
			url:          "http://go.example.com/promo?ref=mail",
			wantLocation: "https://example.com/landing?lang=en&ref=mail",
		},
		{
			name:         "drops query string when disabled",
			targetURL:    "https://example.com/landing",
			forwardQuery: false,
			url:          "http://go.example.com/promo?ref=mail",
			wantLocation: "https://example.com/landing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeRedirectStore()
			store.addDomain("go.example.com", "").shortURLs["promo"] = &handlers.RedirectShortURL{
				PublicID:     "su_1",
				TargetURL:    tc.targetURL,
				ForwardQuery: tc.forwardQuery,
			}

			resp := doRedirectRequest(setupRedirectMux(store), tc.url, nil)

			assertRedirect(t, resp, tc.wantLocation)
		})
	}
}

func TestRedirectPlatformSpecific(t *testing.T) {
	const (
		iPhoneUA  = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
		iPadUA    = "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
		androidUA = "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"
		macUA     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
		windowsUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
		linuxUA   = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	)

	platformURLs := []handlers.RedirectPlatformURL{
		{Platform: "iOS", TargetURL: "https://example.com/ios"},
		{Platform: "Android", TargetURL: "https://example.com/android"},
		{Platform: "Windows", TargetURL: "https://example.com/windows"},
	}

	cases := []struct {
		name         string
		userAgent    string
		platformURLs []handlers.RedirectPlatformURL
		wantLocation string
	}{
		{
			name:         "iPhone gets iOS URL",
			userAgent:    iPhoneUA,
			platformURLs: platformURLs,
			wantLocation: "https://example.com/ios",
		},
		{
			name:         "Android gets Android URL",
			userAgent:    androidUA,
			platformURLs: platformURLs,
			wantLocation: "https://example.com/android",
		},
		{
			name:         "Windows gets Windows URL",
			userAgent:    windowsUA,
			platformURLs: platformURLs,
			wantLocation: "https://example.com/windows",
		},
		{
			name:         "platform without specific URL gets default target",
			userAgent:    macUA,
			platformURLs: platformURLs,
			wantLocation: "https://example.com/landing",
		},
		{
			name:      "macOS gets macOS URL",
			userAgent: macUA,
			platformURLs: []handlers.RedirectPlatformURL{
				{Platform: "macOS", TargetURL: "https://example.com/mac"},
			},
			wantLocation: "https://example.com/mac",
		},
		{
			name:         "iPad gets iOS URL",
			userAgent:    iPadUA,
			platformURLs: platformURLs,
			wantLocation: "https://example.com/ios",
		},
		{
			name:      "iPhone matches Mobile category when no iOS URL exists",
			userAgent: iPhoneUA,
			platformURLs: []handlers.RedirectPlatformURL{
				{Platform: "Mobile", TargetURL: "https://example.com/mobile"},
			},
			wantLocation: "https://example.com/mobile",
		},
		{
			name:      "iPad matches Mobile category when no iOS URL exists",
			userAgent: iPadUA,
			platformURLs: []handlers.RedirectPlatformURL{
				{Platform: "Mobile", TargetURL: "https://example.com/mobile"},
			},
			wantLocation: "https://example.com/mobile",
		},
		{
			name:      "Linux matches Desktop category",
			userAgent: linuxUA,
			platformURLs: []handlers.RedirectPlatformURL{
				{Platform: "Desktop", TargetURL: "https://example.com/desktop"},
			},
			wantLocation: "https://example.com/desktop",
		},
		{
			name:      "Android does not match Linux despite Linux in user agent",
			userAgent: androidUA,
			platformURLs: []handlers.RedirectPlatformURL{
				{Platform: "Linux", TargetURL: "https://example.com/linux"},
			},
			wantLocation: "https://example.com/landing",
		},
		{
			name:         "no user agent gets default target",
			userAgent:    "",
			platformURLs: platformURLs,
			wantLocation: "https://example.com/landing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeRedirectStore()
			store.addDomain("go.example.com", "").shortURLs["promo"] = &handlers.RedirectShortURL{
				PublicID:     "su_1",
				TargetURL:    "https://example.com/landing",
				PlatformURLs: tc.platformURLs,
			}

			header := http.Header{}
			if tc.userAgent != "" {
				header.Set("User-Agent", tc.userAgent)
			}
			resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/promo", header)

			assertRedirect(t, resp, tc.wantLocation)
		})
	}
}

func TestRedirectExpiredPlatformURLUsesPlatformFallback(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)

	store := newFakeRedirectStore()
	store.addDomain("go.example.com", "https://example.com/not-found").shortURLs["promo"] = &handlers.RedirectShortURL{
		PublicID:    "su_1",
		TargetURL:   "https://example.com/landing",
		FallbackURL: "https://example.com/expired",
		ValidUntil:  &past,
		PlatformURLs: []handlers.RedirectPlatformURL{
			{Platform: "Android", TargetURL: "https://example.com/android", FallbackURL: "https://example.com/android-expired"},
		},
	}

	header := http.Header{}
	header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36")
	resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/promo", header)

	assertRedirect(t, resp, "https://example.com/android-expired")
}

func TestRedirectRecordsVisit(t *testing.T) {
	store := newFakeRedirectStore()
	store.addDomain("go.example.com", "").shortURLs["promo"] = &handlers.RedirectShortURL{
		PublicID:     "su_1",
		TargetURL:    "https://example.com/landing",
		ForwardQuery: true,
	}

	header := http.Header{}
	header.Set("User-Agent", "test-agent/1.0")
	header.Set("Referer", "https://social.example.com/post/42")
	header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/promo?utm_campaign=spring", header)

	assertRedirect(t, resp, "https://example.com/landing?utm_campaign=spring")

	if len(store.visits) != 1 {
		t.Fatalf("expected 1 visit, got %d", len(store.visits))
	}
	visit := store.visits[0]
	if visit.ShortURLPublicID != "su_1" {
		t.Errorf("expected short URL public ID %q, got %q", "su_1", visit.ShortURLPublicID)
	}
	if visit.IPAddress != "203.0.113.7" {
		t.Errorf("expected IP address %q, got %q", "203.0.113.7", visit.IPAddress)
	}
	if visit.UserAgent != "test-agent/1.0" {
		t.Errorf("expected user agent %q, got %q", "test-agent/1.0", visit.UserAgent)
	}
	if visit.Referer != "https://social.example.com/post/42" {
		t.Errorf("expected referer %q, got %q", "https://social.example.com/post/42", visit.Referer)
	}
	if visit.Campaign != "spring" {
		t.Errorf("expected campaign %q, got %q", "spring", visit.Campaign)
	}
}

func TestRedirectRecordsVisitWithRemoteAddr(t *testing.T) {
	store := newFakeRedirectStore()
	store.addDomain("go.example.com", "").shortURLs["promo"] = &handlers.RedirectShortURL{
		PublicID:  "su_1",
		TargetURL: "https://example.com/landing",
	}

	// httptest.NewRequest sets RemoteAddr to 192.0.2.1:1234
	resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/promo", nil)

	assertRedirect(t, resp, "https://example.com/landing")

	if len(store.visits) != 1 {
		t.Fatalf("expected 1 visit, got %d", len(store.visits))
	}
	if store.visits[0].IPAddress != "192.0.2.1" {
		t.Errorf("expected IP address %q, got %q", "192.0.2.1", store.visits[0].IPAddress)
	}
}

func TestRedirectRecordsVisitForFallback(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)

	store := newFakeRedirectStore()
	store.addDomain("go.example.com", "").shortURLs["promo"] = &handlers.RedirectShortURL{
		PublicID:    "su_1",
		TargetURL:   "https://example.com/landing",
		FallbackURL: "https://example.com/expired",
		ValidUntil:  &past,
	}

	resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/promo", nil)

	assertRedirect(t, resp, "https://example.com/expired")

	if len(store.visits) != 1 {
		t.Fatalf("expected 1 visit for fallback redirect, got %d", len(store.visits))
	}
}

func TestRedirectVisitFailureStillRedirects(t *testing.T) {
	var logBuf bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	store := newFakeRedirectStore()
	store.visitErr = fmt.Errorf("database is down")
	store.addDomain("go.example.com", "").shortURLs["promo"] = &handlers.RedirectShortURL{
		PublicID:  "su_1",
		TargetURL: "https://example.com/landing",
	}

	resp := doRedirectRequest(setupRedirectMux(store), "http://go.example.com/promo", nil)

	assertRedirect(t, resp, "https://example.com/landing")

	if !strings.Contains(logBuf.String(), "Failed to record visit") {
		t.Errorf("expected visit failure to be logged, got: %s", logBuf.String())
	}
}
