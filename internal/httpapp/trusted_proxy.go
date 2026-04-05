package httpapp

import (
	"net"
	"net/http"
	"strings"
)

// parseTrustedProxyNets parses a comma-separated list of IPs or CIDRs (e.g. "127.0.0.1,10.0.0.0/8").
func parseTrustedProxyNets(raw string) (nets []*net.IPNet, singles []net.IP) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			_, ipnet, err := net.ParseCIDR(part)
			if err == nil && ipnet != nil {
				nets = append(nets, ipnet)
			}
			continue
		}
		if ip := net.ParseIP(part); ip != nil {
			singles = append(singles, ip)
		}
	}
	return nets, singles
}

func ipTrusted(ip net.IP, nets []*net.IPNet, singles []net.IP) bool {
	if ip == nil {
		return false
	}
	for _, s := range singles {
		if s.Equal(ip) {
			return true
		}
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// forwardedClientIP returns the leftmost IP in X-Forwarded-For (original client behind proxies).
func forwardedClientIP(xff string) string {
	xff = strings.TrimSpace(xff)
	if xff == "" {
		return ""
	}
	parts := strings.Split(xff, ",")
	first := strings.TrimSpace(parts[0])
	if host, _, err := net.SplitHostPort(first); err == nil {
		first = host
	}
	return first
}

// trustedProxyRealIP overwrites r.RemoteAddr with the forwarded client when the direct peer is trusted.
func trustedProxyRealIP(nets []*net.IPNet, singles []net.IP) func(http.Handler) http.Handler {
	if len(nets) == 0 && len(singles) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			directIP := net.ParseIP(host)
			if directIP != nil && ipTrusted(directIP, nets, singles) {
				if client := forwardedClientIP(r.Header.Get("X-Forwarded-For")); client != "" {
					if ip := net.ParseIP(client); ip != nil {
						r.RemoteAddr = net.JoinHostPort(ip.String(), "0")
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
