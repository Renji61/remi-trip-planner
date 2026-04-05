package httpapp

import (
	"context"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"remi-trip-planner/internal/trips"
)

type ctxKey int

const (
	ctxKeyUserID ctxKey = iota + 1
	ctxKeySessionRaw
	ctxKeyCSRF
	ctxKeyUser
	ctxKeyTripAccess
)

const sessionCookieName = "remi_session"
const sessionCookieMaxAge = 30 * 24 * 60 * 60

func contextWithUser(ctx context.Context, u trips.User, sessionRaw, csrf string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, u.ID)
	ctx = context.WithValue(ctx, ctxKeyUser, u)
	ctx = context.WithValue(ctx, ctxKeySessionRaw, sessionRaw)
	ctx = context.WithValue(ctx, ctxKeyCSRF, csrf)
	return ctx
}

// CurrentUserID returns the logged-in user id or empty.
func CurrentUserID(ctx context.Context) string {
	s, _ := ctx.Value(ctxKeyUserID).(string)
	return s
}

// CurrentUser returns the logged-in user or zero value.
func CurrentUser(ctx context.Context) trips.User {
	u, _ := ctx.Value(ctxKeyUser).(trips.User)
	return u
}

// SessionTokenRaw returns the raw session cookie value (for logout).
func SessionTokenRaw(ctx context.Context) string {
	s, _ := ctx.Value(ctxKeySessionRaw).(string)
	return s
}

// CSRFToken from session for forms.
func CSRFToken(ctx context.Context) string {
	s, _ := ctx.Value(ctxKeyCSRF).(string)
	return s
}

func (a *app) withSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookieName)
		if err != nil || c.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		u, sess, err := a.tripService.SessionUser(r.Context(), c.Value)
		if err != nil {
			http.SetCookie(w, a.clearSessionCookie())
			next.ServeHTTP(w, r)
			return
		}
		r = r.WithContext(contextWithUser(r.Context(), u, c.Value, sess.CSRFToken))
		next.ServeHTTP(w, r)
	})
}

func (a *app) clearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.env.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	}
}

func (a *app) writeSessionCookie(w http.ResponseWriter, rawToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    rawToken,
		Path:     "/",
		MaxAge:   sessionCookieMaxAge,
		HttpOnly: true,
		Secure:   a.env.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

func csrfTokenFromRequest(r *http.Request) (string, error) {
	if r == nil {
		return "", nil
	}
	if got := strings.TrimSpace(r.Header.Get("X-CSRF-Token")); got != "" {
		return got, nil
	}
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/json") || strings.HasPrefix(r.URL.Path, "/api/") {
		return "", nil
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if err2 := r.ParseForm(); err2 != nil {
			return "", err2
		}
	}
	return strings.TrimSpace(r.FormValue("csrf_token")), nil
}

func requestHasValidCSRF(r *http.Request) (bool, error) {
	got, err := csrfTokenFromRequest(r)
	if err != nil {
		return false, err
	}
	want := CSRFToken(r.Context())
	if want == "" {
		return false, nil
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1, nil
}

func (a *app) requireRegisteredUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := a.tripService.CountUsers(r.Context())
		if err != nil {
			writeInternalServerError(w, r, err)
			return
		}
		if n == 0 {
			http.Redirect(w, r, "/setup", http.StatusSeeOther)
			return
		}
		uid := CurrentUserID(r.Context())
		if uid == "" {
			q := url.QueryEscape(r.URL.RequestURI())
			http.Redirect(w, r, "/login?next="+q, http.StatusSeeOther)
			return
		}
		_ = a.tripService.EnsureUserSettings(r.Context(), uid)
		next.ServeHTTP(w, r)
	})
}

func (a *app) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := CurrentUser(r.Context())
		if u.ID == "" || !u.IsAdmin {
			http.Error(w, "Administrator access is required.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *app) verifyCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		p := r.URL.Path
		if p == "/login" || p == "/setup" || p == "/register" {
			next.ServeHTTP(w, r)
			return
		}
		ok, err := requestHasValidCSRF(r)
		if err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if !ok {
			http.Error(w, "invalid security token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *app) requireTripAccess(w http.ResponseWriter, r *http.Request, tripID string) (trips.TripAccess, bool) {
	uid := CurrentUserID(r.Context())
	acc, err := a.tripService.TripAccess(r.Context(), tripID, uid)
	if err != nil {
		if errors.Is(err, trips.ErrAuthRequired) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return acc, false
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return acc, false
	}
	return acc, true
}

func (a *app) requireTripOwner(w http.ResponseWriter, r *http.Request, tripID string) bool {
	acc, ok := a.requireTripAccess(w, r, tripID)
	if !ok {
		return false
	}
	if !acc.IsOwner {
		http.Error(w, "only the trip owner can do this", http.StatusForbidden)
		return false
	}
	return true
}

// TripAccessFromContext is set by tripIDAccessMiddleware for /trips/{tripID}/* routes.
func TripAccessFromContext(ctx context.Context) (trips.TripAccess, bool) {
	acc, ok := ctx.Value(ctxKeyTripAccess).(trips.TripAccess)
	return acc, ok
}

// absoluteOrigin returns the public base URL (scheme + host) for building invite links.
func absoluteOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if p := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))); p == "https" || p == "http" {
		scheme = p
	}
	host := sanitizeExternalHost(strings.TrimSpace(r.Host))
	if xh := sanitizeExternalHost(strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))); xh != "" {
		host = xh
	}
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host
}

// sanitizeExternalHost normalizes a possibly forwarded host and rejects unsafe values.
func sanitizeExternalHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Proxies can append multiple values; use the first hop only.
	if i := strings.Index(raw, ","); i >= 0 {
		raw = strings.TrimSpace(raw[:i])
	}
	// Disallow host values that can break URL semantics or be abused in links.
	if strings.ContainsAny(raw, "/\\@?&# \t\r\n") {
		return ""
	}
	// Validate as host[:port] (or bracketed IPv6 host[:port]).
	if h, p, err := net.SplitHostPort(raw); err == nil {
		if h == "" || strings.ContainsAny(h, "/\\@?&# \t\r\n") || p == "" {
			return ""
		}
		return raw
	}
	if strings.Contains(raw, ":") && !strings.HasPrefix(raw, "[") {
		return ""
	}
	return raw
}

func (a *app) tripIDAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tripID := chi.URLParam(r, "tripID")
		if tripID == "" {
			http.NotFound(w, r)
			return
		}
		acc, ok := a.requireTripAccess(w, r, tripID)
		if !ok {
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyTripAccess, acc)))
	})
}
