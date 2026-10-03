// Package devicetoken implements the device token domain: the push
// notification token registered by a user's device. It maps to the
// device_tokens table.
package devicetoken

import "time"

// Platform values a DeviceToken may hold.
const (
	PlatformAndroid = "ANDROID"
	PlatformIOS     = "IOS"
	PlatformWeb     = "WEB"
)

// DeviceToken is the domain entity.
type DeviceToken struct {
	ID        string
	UserID    string
	Token     string
	Platform  string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateInput is the data required to register a new device token.
type CreateInput struct {
	UserID   string `json:"user_id" validate:"required,uuid"`
	Token    string `json:"token" validate:"required,max=512"`
	Platform string `json:"platform" validate:"required,oneof=ANDROID IOS WEB"`
}
