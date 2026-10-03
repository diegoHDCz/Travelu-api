package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

// Middleware is the standard net/http middleware signature used across the
// whole application.
type Middleware func(http.Handler) http.Handler

// Recover catches panics in downstream handlers, logs them and responds with
// a generic 500 instead of letting the process crash.
func Recover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					slog.ErrorContext(r.Context(), "panic recovered",
						"error", rec,
						"request_id", RequestIDFrom(r.Context()),
					)
					WriteJSON(w, http.StatusInternalServerError, errorEnvelope{Error: &Error{
						Code:    "internal_error",
						Message: "an unexpected error occurred",
					}})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

const requestIDHeader = "X-Request-ID"

// RequestID assigns a request ID to every request, reusing the one supplied
// by the client (e.g. forwarded by Caddy) when present, and echoes it back in
// the response header.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(requestIDHeader)
			if id == "" {
				id = newRequestID()
			}
			w.Header().Set(requestIDHeader, id)
			ctx := WithRequestID(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}

// AccessLog logs one structured line per request with method, path, status,
// duration and request ID.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.LogAttrs(r.Context(), slog.LevelInfo, "http_request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", RequestIDFrom(r.Context())),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// CORS applies a configurable allow-list of origins. An empty list disables
// CORS headers entirely; "*" allows any origin.
func CORS(allowedOrigins []string) Middleware {
	allowAll := slices.Contains(allowedOrigins, "*")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || slices.Contains(allowedOrigins, origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodyBytes rejects request bodies larger than limit.
func MaxBodyBytes(limit int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// Chain applies middlewares in the given order, so Chain(a, b, c)(h) runs as
// a(b(c(h))), i.e. a is the outermost layer.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
