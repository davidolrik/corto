package handlers

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	ua "github.com/mileusna/useragent"
)

// RedirectStore defines the interface for resolving short links and recording
// visits.
type RedirectStore interface {
	ResolveRedirect(ctx context.Context, fqdn, slug string) (*RedirectTarget, error)
	RecordVisit(ctx context.Context, v *VisitData) error
}

// RedirectTarget is the result of resolving an incoming host and slug.
type RedirectTarget struct {
	DomainFallbackURL string
	ShortURL          *RedirectShortURL // nil when the slug does not exist on the domain
}

// RedirectShortURL carries everything needed to serve a resolved short link.
type RedirectShortURL struct {
	PublicID     string // public ID of the short_url row, used for visit tracking
	TargetURL    string
	FallbackURL  string
	ForwardQuery bool
	ValidSince   *time.Time
	ValidUntil   *time.Time
	PlatformURLs []RedirectPlatformURL
}

// RedirectPlatformURL is a platform-specific target for a short link.
type RedirectPlatformURL struct {
	Platform    string
	TargetURL   string
	FallbackURL string
}

// VisitData represents a single click on a short link.
type VisitData struct {
	ShortURLPublicID string
	IPAddress        string
	UserAgent        string
	Referer          string
	Country          string
	Campaign         string
}

// RegisterRedirectRoutes registers the public short link redirect endpoint on
// the given mux. It lives outside the Huma API because it serves browsers, not
// API clients.
func RegisterRedirectRoutes(mux *http.ServeMux, store RedirectStore) {
	mux.HandleFunc("GET /{slug}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		host := hostWithoutPort(r.Host)
		slug := r.PathValue("slug")

		target, err := store.ResolveRedirect(ctx, host, slug)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		shortURL := target.ShortURL
		if shortURL == nil {
			if target.DomainFallbackURL != "" {
				http.Redirect(w, r, target.DomainFallbackURL, http.StatusFound)
				return
			}
			http.NotFound(w, r)
			return
		}

		// Pick the platform-specific URL matching the visitor, if any
		targetURL := shortURL.TargetURL
		fallbackURL := shortURL.FallbackURL
		if platformURL := matchPlatformURL(shortURL.PlatformURLs, r.UserAgent()); platformURL != nil {
			targetURL = platformURL.TargetURL
			if platformURL.FallbackURL != "" {
				fallbackURL = platformURL.FallbackURL
			}
		}

		// Outside the validity window the fallback chain applies:
		// platform fallback, short code fallback, domain fallback
		if !isValidNow(shortURL.ValidSince, shortURL.ValidUntil) {
			if fallbackURL == "" {
				fallbackURL = target.DomainFallbackURL
			}
			if fallbackURL == "" {
				http.NotFound(w, r)
				return
			}
			recordVisit(ctx, store, shortURL.PublicID, r)
			http.Redirect(w, r, fallbackURL, http.StatusFound)
			return
		}

		if shortURL.ForwardQuery && r.URL.RawQuery != "" {
			targetURL = appendQuery(targetURL, r.URL.RawQuery)
		}

		recordVisit(ctx, store, shortURL.PublicID, r)
		http.Redirect(w, r, targetURL, http.StatusFound)
	})
}

// recordVisit stores a visit for the redirect. Failures are logged but never
// block the redirect itself.
func recordVisit(ctx context.Context, store RedirectStore, shortURLPublicID string, r *http.Request) {
	visit := &VisitData{
		ShortURLPublicID: shortURLPublicID,
		IPAddress:        clientIP(r),
		UserAgent:        r.UserAgent(),
		Referer:          r.Referer(),
		Campaign:         r.URL.Query().Get("utm_campaign"),
	}
	if err := store.RecordVisit(ctx, visit); err != nil {
		slog.Warn("Failed to record visit", "short_url", shortURLPublicID, "error", err)
	}
}

// hostWithoutPort strips an optional port from a request host.
func hostWithoutPort(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// clientIP returns the originating client IP, honoring X-Forwarded-For when
// the server runs behind a proxy.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		return strings.TrimSpace(first)
	}
	return hostWithoutPort(r.RemoteAddr)
}

// isValidNow reports whether the current time falls within the optional
// validity window.
func isValidNow(since, until *time.Time) bool {
	now := time.Now()
	if since != nil && now.Before(*since) {
		return false
	}
	if until != nil && now.After(*until) {
		return false
	}
	return true
}

// platformCandidates maps a User-Agent to platform names in order of
// specificity, e.g. an iPhone matches "iOS" before the broader "Mobile".
func platformCandidates(userAgent string) []string {
	parsed := ua.Parse(userAgent)

	var candidates []string
	if parsed.OS != "" {
		candidates = append(candidates, parsed.OS)
	}
	switch {
	case parsed.Mobile, parsed.Tablet:
		candidates = append(candidates, "Mobile")
	case parsed.Desktop:
		candidates = append(candidates, "Desktop")
	}
	return candidates
}

// matchPlatformURL returns the most specific platform URL matching the
// visitor's User-Agent, or nil when none applies.
func matchPlatformURL(platformURLs []RedirectPlatformURL, userAgent string) *RedirectPlatformURL {
	for _, candidate := range platformCandidates(userAgent) {
		for i := range platformURLs {
			if strings.EqualFold(platformURLs[i].Platform, candidate) {
				return &platformURLs[i]
			}
		}
	}
	return nil
}

// appendQuery appends a raw query string to a URL, preserving any query the
// URL already has.
func appendQuery(targetURL, rawQuery string) string {
	u, err := url.Parse(targetURL)
	if err != nil {
		return targetURL
	}
	if u.RawQuery == "" {
		u.RawQuery = rawQuery
	} else {
		u.RawQuery += "&" + rawQuery
	}
	return u.String()
}
