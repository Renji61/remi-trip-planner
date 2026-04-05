package httpapp

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

func authRateLimitRoute(r *http.Request) bool {
	switch {
	case r.Method == http.MethodPost && (r.URL.Path == "/login" || r.URL.Path == "/register" || r.URL.Path == "/setup"):
		return true
	case r.Method == http.MethodGet && r.URL.Path == "/verify-email":
		return strings.TrimSpace(r.URL.Query().Get("token")) != ""
	case r.Method == http.MethodPost && r.URL.Path == "/invites/accept":
		return true
	case r.Method == http.MethodPost && r.URL.Path == "/profile/resend-verify":
		return true
	default:
		return false
	}
}

type authRateLimiter struct {
	mu     sync.Mutex
	byIP   map[string]*rate.Limiter
	limit  rate.Limit
	burst  int
	maxMap int
}

func newAuthRateLimiter(rpm, burst int) *authRateLimiter {
	if rpm < 1 {
		rpm = 40
	}
	if burst < 1 {
		burst = 12
	}
	return &authRateLimiter{
		byIP:   make(map[string]*rate.Limiter),
		limit:  rate.Limit(float64(rpm) / 60.0),
		burst:  burst,
		maxMap: 50_000,
	}
}

func (a *authRateLimiter) allow(ip string) bool {
	if ip == "" {
		ip = "unknown"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.byIP) >= a.maxMap {
		a.byIP = make(map[string]*rate.Limiter)
	}
	l, ok := a.byIP[ip]
	if !ok {
		l = rate.NewLimiter(a.limit, a.burst)
		a.byIP[ip] = l
	}
	return l.Allow()
}

func authRateLimitMiddleware(lim *authRateLimiter) func(http.Handler) http.Handler {
	if lim == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authRateLimitRoute(r) {
				next.ServeHTTP(w, r)
				return
			}
			ip := clientIPString(r.RemoteAddr)
			if !lim.allow(ip) {
				rid, _ := r.Context().Value(ctxKeyRequestID).(string)
				slog.WarnContext(r.Context(), "rate_limit",
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.String("client_ip", ip),
					slog.String("request_id", rid),
				)
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIPString(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return strings.TrimSpace(remoteAddr)
	}
	return host
}
