package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter throttles requests per client IP using an in-memory token
// bucket per visitor. Meant for low-volume, abuse-prone routes like login
// and register, not as a general-purpose rate limiter.
type IPRateLimiter struct {
	mu         sync.Mutex
	visitors   map[string]*visitor
	rps        rate.Limit
	burst      int
	trustProxy bool
}

// NewIPRateLimiter builds a limiter allowing rps requests per second (with
// burst) per IP. When trustProxy is true, the client IP is read from
// X-Forwarded-For (as set by Caddy); otherwise it is read from the raw
// connection, since a spoofed header would otherwise defeat the limiter.
func NewIPRateLimiter(rps float64, burst int, trustProxy bool) *IPRateLimiter {
	return &IPRateLimiter{
		visitors:   make(map[string]*visitor),
		rps:        rate.Limit(rps),
		burst:      burst,
		trustProxy: trustProxy,
	}
}

// Middleware returns an httpx.Middleware enforcing the limit.
func (l *IPRateLimiter) Middleware() httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(l.clientIP(r)) {
				httpx.WriteError(r.Context(), w, errTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *IPRateLimiter) clientIP(r *http.Request) string {
	if l.trustProxy {
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

func (l *IPRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.evictStaleLocked()

	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()

	return v.limiter.Allow()
}

// evictStaleLocked drops visitors idle for more than 10 minutes so the map
// does not grow without bound. Must be called with mu held.
func (l *IPRateLimiter) evictStaleLocked() {
	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, v := range l.visitors {
		if v.lastSeen.Before(cutoff) {
			delete(l.visitors, ip)
		}
	}
}
