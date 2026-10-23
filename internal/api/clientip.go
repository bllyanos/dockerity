package api

import (
	"net"
	"net/http"
	"strings"
)

// clientIP returns the caller's address. Dockerity sits behind exactly one
// proxy (Traefik), so the rightmost X-Forwarded-For entry is the one Traefik
// appended and cannot be spoofed by the client.
func clientIP(r *http.Request) string {
	if xff := r.Header.Values("X-Forwarded-For"); len(xff) > 0 {
		last := xff[len(xff)-1]
		if i := strings.LastIndexByte(last, ','); i >= 0 {
			last = last[i+1:]
		}
		if ip := strings.TrimSpace(last); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
