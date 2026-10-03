// Package auth implements authentication: access tokens (JWT), opaque
// refresh tokens with rotation and reuse detection, the RequireAuth
// middleware and the HTTP handlers for register/login/refresh/logout.
package auth

import "context"

type contextKey int

const (
	userIDKey contextKey = iota
	deviceTokenKey
)

// WithUserID returns a context carrying the authenticated user's ID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFrom extracts the authenticated user ID set by RequireAuth.
func UserIDFrom(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// WithDeviceToken returns a context carrying the device token carried by the
// request's access token, if any.
func WithDeviceToken(ctx context.Context, deviceToken string) context.Context {
	return context.WithValue(ctx, deviceTokenKey, deviceToken)
}

// DeviceTokenFrom extracts the device token set by RequireAuth. ok is false
// if the access token did not carry one.
func DeviceTokenFrom(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(deviceTokenKey).(string)
	return token, ok && token != ""
}
