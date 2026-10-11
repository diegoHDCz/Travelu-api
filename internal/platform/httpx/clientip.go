package httpx

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns the caller's IP. When trustProxy is true it is read from
// X-Forwarded-For (as set by a trusted reverse proxy); otherwise from the raw
// connection, since a client-supplied header could be spoofed. Behind Lambda
// httpadapter sets r.RemoteAddr from API Gateway's request metadata, and a
// bare IP (no port) is returned as-is.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			if idx := strings.Index(fwd, ","); idx != -1 {
				fwd = fwd[:idx]
			}
			return strings.TrimSpace(fwd)
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
