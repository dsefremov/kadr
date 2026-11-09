package httpx

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns the client address for rate limits (X-Forwarded-For first hop or RemoteAddr).
func ClientIP(r *http.Request) net.IP {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if parsed := net.ParseIP(ip); parsed != nil {
			return parsed
		}
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host == "" {
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}
