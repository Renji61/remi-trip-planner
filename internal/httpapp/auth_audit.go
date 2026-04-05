package httpapp

import (
	"net/http"
	"strings"
)

const maxUserAgentLogLen = 512

// TruncatedUserAgent returns User-Agent trimmed for structured logs (no PII beyond what browsers send).
func TruncatedUserAgent(r *http.Request) string {
	if r == nil {
		return ""
	}
	ua := strings.TrimSpace(r.Header.Get("User-Agent"))
	if len(ua) > maxUserAgentLogLen {
		return ua[:maxUserAgentLogLen]
	}
	return ua
}
