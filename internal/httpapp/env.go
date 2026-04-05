package httpapp

import (
	"os"
	"strconv"
	"strings"
)

// RemiEnv holds deployment flags derived from environment (see README).
type RemiEnv struct {
	Production    bool
	SecureCookies bool
	// HSTSMaxAge is sent as max-age when Production is true (0 disables HSTS).
	HSTSMaxAge int
	// TrustedProxies: CIDRs or single IPs; when the direct client IP matches, X-Forwarded-For is honored for client IP (rate limits, logs).
	TrustedProxies string
	// RateLimitAuthRPM is per-IP requests per minute for sensitive auth-related routes.
	RateLimitAuthRPM int
	RateLimitBurst   int
	// HealthzCheckDB when true makes GET /healthz ping SQLite.
	HealthzCheckDB bool
}

// LoadRemiEnv reads REMI_ENV=production and related variables.
func LoadRemiEnv() RemiEnv {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("REMI_ENV")))
	prod := env == "production"
	rpm := envIntPositive("REMI_RATE_LIMIT_AUTH_RPM", 40)
	burst := envIntPositive("REMI_RATE_LIMIT_AUTH_BURST", 12)
	if burst < 1 {
		burst = 12
	}
	hsts := envIntPositive("REMI_HSTS_MAX_AGE", 31536000)
	return RemiEnv{
		Production:       prod,
		SecureCookies:    prod,
		HSTSMaxAge:       hsts,
		TrustedProxies:   strings.TrimSpace(os.Getenv("REMI_TRUSTED_PROXIES")),
		RateLimitAuthRPM: rpm,
		RateLimitBurst:   burst,
		HealthzCheckDB:   envTruthy(os.Getenv("REMI_HEALTHZ_DB")),
	}
}

func envIntPositive(key string, defaultVal int) int {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}

func envTruthy(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}
