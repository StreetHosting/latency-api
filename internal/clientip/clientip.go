package clientip

import (
	"net"
	"net/http"
	"strings"
)

// FromRequest returns the client IP set by nginx (X-Real-IP). Falls back to the
// first public IP in X-Forwarded-For only when X-Real-IP is absent.
func FromRequest(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" && validTarget(ip) {
		return ip
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return ""
	}
	for _, part := range strings.Split(xff, ",") {
		ip := strings.TrimSpace(part)
		if validTarget(ip) {
			return ip
		}
	}
	return ""
}

func validTarget(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	if parsed.IsLoopback() || parsed.IsUnspecified() || parsed.IsLinkLocalUnicast() {
		return false
	}
	// Block multicast; allow private IPs (lab/VPN clients).
	if parsed.IsMulticast() {
		return false
	}
	return true
}
