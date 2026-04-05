package httpapp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"remi-trip-planner/internal/trips"
)

func TestLoadRemiEnvProductionSecureCookies(t *testing.T) {
	t.Setenv("REMI_ENV", "production")
	env := LoadRemiEnv()
	if !env.Production || !env.SecureCookies {
		t.Fatalf("expected production secure cookies, got %+v", env)
	}
	t.Setenv("REMI_ENV", "development")
	env2 := LoadRemiEnv()
	if env2.Production || env2.SecureCookies {
		t.Fatalf("expected dev defaults, got %+v", env2)
	}
}

func TestWriteSessionCookieSecureInProduction(t *testing.T) {
	t.Setenv("REMI_ENV", "production")
	env := LoadRemiEnv()
	a := &app{env: env}
	rr := httptest.NewRecorder()
	a.writeSessionCookie(rr, "session-token-value")
	resp := rr.Result()
	defer resp.Body.Close()
	var saw bool
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			saw = true
			if !c.Secure {
				t.Fatal("expected Secure cookie in production")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Fatalf("expected SameSite=Strict, got %v", c.SameSite)
			}
		}
	}
	if !saw {
		t.Fatal("session cookie not set")
	}
}

func TestVerifyCSRFAcceptsJSONHeader(t *testing.T) {
	a := &app{}
	h := a.verifyCSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-1/sync", bytes.NewBufferString(`{"ops":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "csrf-123")
	req = req.WithContext(contextWithUser(req.Context(), trips.User{ID: "user-1"}, "session-1", "csrf-123"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected json csrf header to pass, got %d", rr.Code)
	}
}

func TestVerifyCSRFRejectsMissingJSONHeader(t *testing.T) {
	a := &app{}
	h := a.verifyCSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/trip-1/sync", bytes.NewBufferString(`{"ops":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithUser(req.Context(), trips.User{ID: "user-1"}, "session-1", "csrf-123"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected missing json csrf header to fail, got %d", rr.Code)
	}
}

func TestRemiRequestIDHeader(t *testing.T) {
	h := remiRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Error("missing request id in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID response header")
	}
}

func TestAuthRateLimitSecondRequest429(t *testing.T) {
	lim := newAuthRateLimiter(1, 1)
	mw := authRateLimitMiddleware(lim)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := mw(inner)

	do := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "198.51.100.22:5555"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr
	}

	if do().Code != http.StatusOK {
		t.Fatal("first request should pass")
	}
	if do().Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be 429, got %d", do().Code)
	}
}

func TestSecurityHeadersHSTSInProduction(t *testing.T) {
	env := RemiEnv{Production: true, HSTSMaxAge: 60}
	h := securityHeaders(env)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rr.Header().Get("Strict-Transport-Security"); got != "max-age=60" {
		t.Fatalf("HSTS: got %q", got)
	}
	if rr.Header().Get("Content-Security-Policy-Report-Only") == "" {
		t.Fatal("expected CSP-Report-Only in production")
	}
}

func TestSecurityHeadersNoHSTSWhenDev(t *testing.T) {
	env := RemiEnv{Production: false}
	h := securityHeaders(env)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("unexpected HSTS in dev")
	}
}
