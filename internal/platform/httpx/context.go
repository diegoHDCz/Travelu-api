package httpx

import "context"

type contextKey int

const requestIDKey contextKey = iota

// WithRequestID returns a context carrying the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFrom extracts the request ID set by the RequestID middleware, if any.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
