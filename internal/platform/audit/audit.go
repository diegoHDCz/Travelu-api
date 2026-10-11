// Package audit records one structured "audit" log line per API request:
// who did it (actor, IP, user agent), what they did (action and target
// resource), and how it ended (status, error code). Lines carry
// log_type=audit so they can be filtered apart from the regular application
// logs in CloudWatch Logs Insights, e.g.
//
//	filter log_type = "audit" and actor_id = "<user id>"
//
// The Middleware owns the entry for the whole request; inner layers enrich
// it through the context: Action names the route, RequireAuth sets the
// actor, handlers set the resource ID of what they created, services add
// extra attributes, and httpx.WriteError records the error code.
//
// Never put secrets (passwords, tokens) or request bodies in an entry.
package audit

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type contextKey struct{}

// entry is the mutable audit record for a single request. A request is
// served by a single goroutine, so it needs no locking.
type entry struct {
	action     string
	route      string
	actorID    string
	resourceID string
	errorCode  string
	attrs      []slog.Attr
}

func from(ctx context.Context) *entry {
	e, _ := ctx.Value(contextKey{}).(*entry)
	return e
}

// Config tells the Middleware how to read request metadata owned by other
// packages, keeping audit free of internal imports.
type Config struct {
	Logger    *slog.Logger
	RequestID func(context.Context) string
	ClientIP  func(*http.Request) string
	// Skip excludes noisy, non-sensitive paths (health checks, docs).
	Skip func(*http.Request) bool
}

// Middleware emits the audit line once the request has been served, even if
// a downstream handler panics.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			e := &entry{}
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			defer func() {
				if rec := recover(); rec != nil {
					sw.status = http.StatusInternalServerError
					e.emit(r, cfg, sw.status, start)
					panic(rec)
				}
				e.emit(r, cfg, sw.status, start)
			}()

			next.ServeHTTP(sw, r.WithContext(context.WithValue(r.Context(), contextKey{}, e)))
		})
	}
}

func (e *entry) emit(r *http.Request, cfg Config, status int, start time.Time) {
	action := e.action
	if action == "" {
		action = "unmatched_route"
	}

	attrs := []slog.Attr{
		slog.String("log_type", "audit"),
		slog.String("action", action),
		slog.String("outcome", outcome(status)),
		slog.Int("status", status),
		slog.String("actor_id", e.actorID),
		slog.String("resource_id", e.resourceID),
		slog.String("error_code", e.errorCode),
		slog.String("method", r.Method),
		slog.String("route", e.route),
		slog.String("path", r.URL.Path),
		slog.String("ip", cfg.ClientIP(r)),
		slog.String("user_agent", r.UserAgent()),
		slog.String("request_id", cfg.RequestID(r.Context())),
		slog.Duration("duration", time.Since(start)),
	}
	attrs = append(attrs, e.attrs...)

	cfg.Logger.LogAttrs(r.Context(), slog.LevelInfo, "audit", attrs...)
}

func outcome(status int) string {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return "denied"
	case status >= 400:
		return "failure"
	default:
		return "success"
	}
}

// Action wraps a route's handler so its audit entry is named action (e.g.
// "booking.create") and carries the matched route pattern and, when the
// route has an {id} or {slug} wildcard, the resource ID. Register it outside any auth
// middleware so rejected attempts are labeled too.
func Action(action string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if e := from(r.Context()); e != nil {
			e.action = action
			e.route = r.Pattern
			if id := r.PathValue("id"); id != "" {
				e.resourceID = id
			} else if slug := r.PathValue("slug"); slug != "" {
				e.resourceID = slug
			}
		}
		next.ServeHTTP(w, r)
	})
}

// SetActor records the ID of the user performing the request.
func SetActor(ctx context.Context, userID string) {
	if e := from(ctx); e != nil {
		e.actorID = userID
	}
}

// SetResourceID records the ID of the resource the request acted on, for
// routes where it is only known after the fact (creates).
func SetResourceID(ctx context.Context, id string) {
	if e := from(ctx); e != nil {
		e.resourceID = id
	}
}

// SetErrorCode records the machine-readable error code sent to the client.
func SetErrorCode(ctx context.Context, code string) {
	if e := from(ctx); e != nil {
		e.errorCode = code
	}
}

// Add attaches extra attributes to the request's audit line, e.g. the
// internal reason behind a deliberately generic client error.
func Add(ctx context.Context, attrs ...slog.Attr) {
	if e := from(ctx); e != nil {
		e.attrs = append(e.attrs, attrs...)
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
